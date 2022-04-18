package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func profileUpdater(database store.PersonStore) {
	var update = func() {
		localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		people, errGetExpired := database.GetExpiredProfiles(localCtx, 100)
		if errGetExpired != nil {
			log.Errorf("Failed to get expired profiles: %v", errGetExpired)
			return
		}
		var sids steamid.Collection
		for _, person := range people {
			sids = append(sids, person.SteamID)
		}
		summaries, errSummaries := steamweb.PlayerSummaries(sids)
		if errSummaries != nil {
			log.Errorf("Failed to get Player summaries: %v", errSummaries)
			return
		}
		for _, summary := range summaries {
			// TODO batch update upserts
			sid, errSid := steamid.SID64FromString(summary.Steamid)
			if errSid != nil {
				log.Errorf("Failed to parse steamid from webapi: %v", errSid)
				continue
			}
			person := model.NewPerson(sid)
			if errGetPerson := database.GetOrCreatePersonBySteamID(localCtx, sid, &person); errGetPerson != nil {
				log.Errorf("Failed to get person: %v", errGetPerson)
				continue
			}
			person.PlayerSummary = &summary
			if errSavePerson := database.SavePerson(localCtx, &person); errSavePerson != nil {
				log.Errorf("Failed to save person: %v", errSavePerson)
				continue
			}
		}
		log.WithFields(log.Fields{"count": len(summaries)}).Trace("Profiles updated")
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			log.Debugf("profileUpdater shutting down")
			return
		}
	}
}

// serverStateUpdater concurrently ( num_servers * 2) updates all known servers' A2S and rcon status
// information. This data is accessed often so it is cached
func serverStateUpdater(database store.ServerStore) {
	freq, errDuration := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errDuration)
	}
	var update = func(ctx context.Context) (model.ServerStateCollection, error) {
		servers, errGetServers := database.GetServers(ctx, false)
		if errGetServers != nil {
			return nil, errors.Wrapf(errGetServers, "Failed to get servers")
		}

		newServers := model.ServerStateCollection{}
		done := make(chan any)
		results := make(chan model.ServerState)

		updateWaitGroup := &sync.WaitGroup{}
		updateWaitGroup.Add(1)
		go func() {
			defer updateWaitGroup.Done()
			for {
				select {
				case <-done:
					return
				case r := <-results:
					newServers[r.Name] = r
				}
			}
		}()
		waitGroup := &sync.WaitGroup{}
		for _, server := range servers {
			waitGroup.Add(1)
			go func(server model.Server) {
				defer waitGroup.Done()
				serverState := model.ServerState{}
				serverState.Region = server.Region
				serverState.Enabled = server.IsEnabled
				serverState.CountryCode = server.CC
				serverState.Name = server.ServerName
				serverState.Reserved = 8
				queryWaitGroup := &sync.WaitGroup{}
				queryWaitGroup.Add(2)
				go func(state *model.ServerState) {
					defer queryWaitGroup.Done()
					status, errS := query.GetServerStatus(server)
					if errS != nil {
						log.Tracef("Failed to update server status: %v", errS)
						return
					}
					serverState.Status = status
				}(&serverState)
				go func(state *model.ServerState) {
					defer queryWaitGroup.Done()
					a, errA := query.A2SQueryServer(server)
					if errA != nil {
						log.Tracef("Failed to update a2s status: %v", errA)
						return
					}
					serverState.A2S = *a
					playerCounter.With(prometheus.Labels{"server_name": server.ServerName}).
						Observe(float64(a.Players))
					if a.Players > 1 {
						mapCounter.With(prometheus.Labels{"map": a.Map}).Add(freq.Seconds())
					}
				}(&serverState)
				queryWaitGroup.Wait()
				serverState.LastUpdate = config.Now()
				results <- serverState
			}(server)
		}
		waitGroup.Wait()
		close(done)
		updateWaitGroup.Wait()
		log.WithFields(log.Fields{"count": len(servers)}).Tracef("Servers updated")
		c := model.ServerStateCollection{}
		for k, v := range newServers {
			c[k] = v
		}
		return c, nil
	}
	newServerState, errNewServerState := update(ctx)
	if errNewServerState != nil {
		log.Errorf("Failed to update servers: %v", errNewServerState)
	} else {
		serversStateMu.Lock()
		serversState = newServerState
		serversStateMu.Unlock()
	}
	ticker := time.NewTicker(freq)
	for {
		select {
		case <-ticker.C:
			newState, errNewState := update(ctx)
			if errNewState != nil {
				log.Errorf("Failed to update servers: %v", errNewState)
			} else {
				serversStateMu.Lock()
				serversState = newState
				serversStateMu.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

// mapChanger watches over servers and checks for servers on maps with 0 players.
// If there is no player for a long enough duration and the map is not one of the
// maps in the default map set, a changelevel request will be made to the server
//
// Relevant config values:
// - general.map_changer_enabled
// - general.default_map
func mapChanger(database store.ServerStore, timeout time.Duration) {
	type at struct {
		lastActive time.Time
		triggered  bool
	}
	activityMap := map[string]*at{}
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			if !config.General.MapChangerEnabled {
				continue
			}
			serversStateMu.RLock()
			stateCopy := ServerState()
			serversStateMu.RUnlock()
			for serverId, state := range stateCopy {
				activity, activityFound := activityMap[serverId]
				if !activityFound || len(state.Status.Players) > 0 {
					activityMap[serverId] = &at{config.Now(), false}
					continue
				}
				if !activity.triggered && time.Since(activity.lastActive) > timeout {
					isDefaultMap := false
					for _, m := range config.General.DefaultMaps {
						if m == stateCopy[serverId].A2S.Map {
							isDefaultMap = true
							break
						}
					}
					if isDefaultMap {
						continue
					}
					var server model.Server
					if errGetServer := database.GetServerByName(context.Background(), serverId, &server); errGetServer != nil {
						log.Errorf("Failed to get server for map changer: %v", errGetServer)
						continue
					}
					nextMap := server.DefaultMap
					if nextMap == "" {
						nextMap = config.General.DefaultMaps[rand.Intn(len(config.General.DefaultMaps))]
					}
					if nextMap == "" {
						log.Errorf("Failed to get valid nextMap value")
						continue
					}
					if server.DefaultMap == state.Status.Map {
						continue
					}
					go func(s model.Server, mapName string) {
						var logger = log.WithFields(log.Fields{"map": nextMap, "reason": "no_activity", "server": serverId})
						logger.Infof("Idle map change triggered")
						if _, errExecRCON := query.ExecRCON(server, fmt.Sprintf("changelevel %s", mapName)); errExecRCON != nil {
							logger.Errorf("failed to exec mapchanger rcon: %v", errExecRCON)
						}
						logger.Infof("Idle map change complete")
					}(server, nextMap)
					activity.triggered = true
					continue
				}
				if activity.triggered {
					activity.triggered = false
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
// TODO save history
func banSweeper(database store.Store) {
	log.WithFields(log.Fields{"service": "ban_sweeper", "status": "ready"}).Debugf("Service status changed")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)
			go func() {
				defer waitGroup.Done()
				expiredBans, errExpiredBans := database.GetExpiredBans(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired expiredBans: %v", errExpiredBans)
				} else {
					for _, expiredBan := range expiredBans {
						if errDrop := database.DropBan(ctx, &expiredBan); errDrop != nil {
							log.Errorf("Failed to drop expired expiredBan: %v", errDrop)
						} else {
							log.Infof("expiredBan expired: %v", expiredBan)
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredNetBans, errExpiredNetBans := database.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired netbans: %v", errExpiredNetBans)
				} else {
					for _, expiredNetBan := range expiredNetBans {
						if errDropBanNet := database.DropBanNet(ctx, &expiredNetBan); errDropBanNet != nil {
							log.Errorf("Failed to drop expired network expiredNetBan: %v", errDropBanNet)
						} else {
							log.Infof("Network expiredNetBan expired: %v", expiredNetBan)
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredASNBans, errExpiredASNBans := database.GetExpiredASNBans(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired asnbans: %v", errExpiredASNBans)
				} else {
					for _, expiredASNBan := range expiredASNBans {
						if errDropASN := database.DropBanASN(ctx, &expiredASNBan); errDropASN != nil {
							log.Errorf("Failed to drop expired asn ban: %v", errDropASN)
						} else {
							log.Infof("ASN ban expired: %v", expiredASNBan)
						}
					}
				}
			}()
			waitGroup.Wait()
		case <-ctx.Done():
			log.Debugf("banSweeper shutting down")
			return
		}
	}
}
