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
func profileUpdater() {
	var update = func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		people, pErr := db.GetExpiredProfiles(ctx, 100)
		if pErr != nil {
			log.Errorf("Failed to get expired profiles: %v", pErr)
			return
		}
		var sids steamid.Collection
		for _, p := range people {
			sids = append(sids, p.SteamID)
		}
		summaries, err2 := steamweb.PlayerSummaries(sids)
		if err2 != nil {
			log.Errorf("Failed to get Player summaries: %v", err2)
			return
		}
		for _, s := range summaries {
			// TODO batch update upserts
			sid, err3 := steamid.SID64FromString(s.Steamid)
			if err3 != nil {
				log.Errorf("Failed to parse steamid from webapi: %v", err3)
				continue
			}
			p := model.NewPerson(sid)
			if err4 := db.GetOrCreatePersonBySteamID(ctx, sid, &p); err4 != nil {
				log.Errorf("Failed to get person: %v", err4)
				continue
			}
			p.PlayerSummary = &s
			if err5 := db.SavePerson(ctx, &p); err5 != nil {
				log.Errorf("Failed to save person: %v", err5)
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
func serverStateUpdater() {
	freq, errD := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errD != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errD)
	}
	var update = func(ctx context.Context) (model.ServerStateCollection, error) {
		servers, err := db.GetServers(ctx, false)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get servers")
		}

		newServers := model.ServerStateCollection{}
		done := make(chan interface{})
		results := make(chan model.ServerState)

		wgC := &sync.WaitGroup{}
		wgC.Add(1)
		go func() {
			defer wgC.Done()
			for {
				select {
				case <-done:
					return
				case r := <-results:
					newServers[r.Name] = r
				}
			}
		}()

		wg := &sync.WaitGroup{}
		for _, srv := range servers {
			wg.Add(1)
			go func(server model.Server) {
				defer wg.Done()
				ss := model.ServerState{}
				ss.Region = server.Region
				ss.Enabled = server.IsEnabled
				ss.CountryCode = server.CC
				ss.Name = server.ServerName
				ss.Reserved = 8
				iwg := &sync.WaitGroup{}
				iwg.Add(2)
				go func(state *model.ServerState) {
					defer iwg.Done()
					status, errS := query.GetServerStatus(server)
					if errS != nil {
						log.Tracef("Failed to update server status: %v", errS)
						return
					}
					ss.Status = status
				}(&ss)
				go func(state *model.ServerState) {
					defer iwg.Done()
					a, errA := query.A2SQueryServer(server)
					if errA != nil {
						log.Tracef("Failed to update a2s status: %v", errA)
						return
					}
					ss.A2S = *a
					playerCountHistogram.With(prometheus.Labels{"server_name": server.ServerName}).
						Observe(float64(a.Players))
					if a.Players > 1 {
						mapCountHistogram.With(prometheus.Labels{"map": a.Map}).Observe(freq.Seconds())
					}
				}(&ss)
				iwg.Wait()
				ss.LastUpdate = time.Now()
				results <- ss
			}(srv)
		}
		wg.Wait()
		close(done)
		wgC.Wait()
		log.WithFields(log.Fields{"count": len(servers)}).Tracef("Servers updated")
		c := model.ServerStateCollection{}
		for k, v := range newServers {
			c[k] = v
		}
		return c, nil
	}
	ns, errNs := update(ctx)
	if errNs != nil {
		log.Errorf("Failed to update servers: %v", errNs)
	} else {
		serversStateMu.Lock()
		serversState = ns
		serversStateMu.Unlock()
	}
	ticker := time.NewTicker(freq)
	for {
		select {
		case <-ticker.C:
			ns, errNs := update(ctx)
			if errNs != nil {
				log.Errorf("Failed to update servers: %v", errNs)
			} else {
				serversStateMu.Lock()
				serversState = ns
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
func mapChanger(timeout time.Duration) {
	type at struct {
		lastActive time.Time
		triggered  bool
	}
	activity := map[string]*at{}
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
				act, found := activity[serverId]
				if !found || len(state.Status.Players) > 0 {
					activity[serverId] = &at{time.Now(), false}
					continue
				}
				if !act.triggered && time.Since(act.lastActive) > timeout {
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
					var srv model.Server
					if err := db.GetServerByName(context.Background(), serverId, &srv); err != nil {
						log.Errorf("Failed to get server for map changer: %v", err)
						continue
					}
					nextMap := srv.DefaultMap
					if nextMap == "" {
						nextMap = config.General.DefaultMaps[rand.Intn(len(config.General.DefaultMaps))]
					}
					if nextMap == "" {
						log.Errorf("Failed to get valid nextMap value")
						continue
					}
					if srv.DefaultMap == state.Status.Map {
						continue
					}
					go func(s model.Server, mapName string) {
						var l = log.WithFields(log.Fields{"map": nextMap, "reason": "no_activity", "srv": serverId})
						l.Infof("Idle map change triggered")
						if _, err := query.ExecRCON(srv, fmt.Sprintf("changelevel %s", mapName)); err != nil {
							l.Errorf("failed to exec mapchanger rcon: %v", err)
						}
						l.Infof("Idle map change complete")
					}(srv, nextMap)
					act.triggered = true
					continue
				}
				if act.triggered {
					act.triggered = false
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func banSweeper() {
	log.WithFields(log.Fields{"service": "ban_sweeper", "status": "ready"}).Debugf("Service status changed")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			wg := &sync.WaitGroup{}
			wg.Add(3)
			go func() {
				defer wg.Done()
				bans, err := db.GetExpiredBans(ctx)
				if err != nil && !errors.Is(err, store.ErrNoResult) {
					log.Warnf("Failed to get expired bans: %v", err)
				} else {
					for _, ban := range bans {
						if err := db.DropBan(ctx, &ban); err != nil {
							log.Errorf("Failed to drop expired ban: %v", err)
						} else {
							log.Infof("ban expired: %v", ban)
						}
					}
				}
			}()
			go func() {
				defer wg.Done()
				netBans, err2 := db.GetExpiredNetBans(ctx)
				if err2 != nil && !errors.Is(err2, store.ErrNoResult) {
					log.Warnf("Failed to get expired netbans: %v", err2)
				} else {
					for _, ban := range netBans {
						if err := db.DropBanNet(ctx, &ban); err != nil {
							log.Errorf("Failed to drop expired network ban: %v", err)
						} else {
							log.Infof("Network ban expired: %v", ban)
						}
					}
				}
			}()
			go func() {
				defer wg.Done()
				asnBans, err3 := db.GetExpiredASNBans(ctx)
				if err3 != nil && !errors.Is(err3, store.ErrNoResult) {
					log.Warnf("Failed to get expired asnbans: %v", err3)
				} else {
					for _, asnBan := range asnBans {
						if err := db.DropBanASN(ctx, &asnBan); err != nil {
							log.Errorf("Failed to drop expired asn ban: %v", err)
						} else {
							log.Infof("ASN ban expired: %v", asnBan)
						}
					}
				}
			}()
			wg.Wait()
		case <-ctx.Done():
			log.Debugf("banSweeper shutting down")
			return
		}
	}
}
