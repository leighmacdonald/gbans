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
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func profileUpdater(ctx context.Context, database store.PersonStore) {
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

func serverA2SStatusUpdater(ctx context.Context, database store.ServerStore, updateFreq time.Duration) {
	var updateStatus = func(localCtx context.Context, localDb store.ServerStore) error {
		cancelCtx, cancel := context.WithTimeout(localCtx, updateFreq/2)
		defer cancel()
		servers, errGetServers := localDb.GetServers(cancelCtx, false)
		if errGetServers != nil {
			return errors.Wrapf(errGetServers, "Failed to get servers")
		}
		waitGroup := &sync.WaitGroup{}
		for _, srv := range servers {
			waitGroup.Add(1)
			go func(server model.Server) {
				defer waitGroup.Done()
				newStatus, errA := query.A2SQueryServer(server)
				if errA != nil {
					log.Tracef("Failed to update a2s status: %v", errA)
					return
				}
				serverStateA2SMu.Lock()
				serverStateA2S[server.ServerNameShort] = *newStatus
				serverStateA2SMu.Unlock()
			}(srv)
		}
		waitGroup.Wait()
		return nil
	}
	ticker := time.NewTicker(updateFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.WithFields(log.Fields{"state": "started"}).Tracef("Server status state updateStatus")
			if errUpdate := updateStatus(ctx, database); errUpdate != nil {
				log.Errorf("Error trying to updateStatus server status state: %v", errUpdate)
			}
			log.WithFields(log.Fields{"state": "success"}).Tracef("Server status state updateStatus")
		}
	}
}

func serverRCONStatusUpdater(ctx context.Context, database store.ServerStore, updateFreq time.Duration) {
	var updateStatus = func(localCtx context.Context, localDb store.ServerStore) error {
		cancelCtx, cancel := context.WithTimeout(localCtx, updateFreq/2)
		defer cancel()
		servers, errGetServers := localDb.GetServers(cancelCtx, false)
		if errGetServers != nil {
			return errors.Wrapf(errGetServers, "Failed to get servers")
		}
		waitGroup := &sync.WaitGroup{}
		for _, srv := range servers {
			waitGroup.Add(1)
			go func(c context.Context, server model.Server) {
				defer waitGroup.Done()
				newStatus, queryErr := query.GetServerStatus(c, server)
				if queryErr != nil {
					log.Tracef("Failed to query server status: %v", queryErr)
					return
				}
				serverStateStatusMu.Lock()
				serverStateStatus[server.ServerNameShort] = newStatus
				serverStateStatusMu.Unlock()
			}(cancelCtx, srv)
		}
		waitGroup.Wait()
		return nil
	}
	ticker := time.NewTicker(updateFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.WithFields(log.Fields{"state": "started"}).Tracef("Server status state updateStatus")
			if errUpdate := updateStatus(ctx, database); errUpdate != nil {
				log.Errorf("Error trying to updateStatus server status state: %v", errUpdate)
			}
			log.WithFields(log.Fields{"state": "success"}).Tracef("Server status state updateStatus")
		}
	}
}

// serverStateRefresher periodically compiles and caches the current known db, rcon & a2s server state
// into a ServerState instance
func serverStateRefresher(ctx context.Context, database store.ServerStore) {
	var refreshState = func() error {
		var newState model.ServerStateCollection
		servers, errServers := database.GetServers(ctx, true)
		if errServers != nil {
			return errors.Errorf("Failed to fetch servers: %v", errServers)
		}
		serverStateA2SMu.RLock()
		defer serverStateA2SMu.RUnlock()
		serverStateStatusMu.RLock()
		defer serverStateStatusMu.RUnlock()
		for _, server := range servers {
			var state model.ServerState
			// use existing state for start?
			state.ServerId = server.ServerID
			state.Name = server.ServerNameLong
			state.NameShort = server.ServerNameShort
			state.Host = server.Address
			state.Port = server.Port
			state.Enabled = server.IsEnabled
			state.Region = server.Region
			state.CountryCode = server.CC
			state.Location = server.Location
			a2sInfo, a2sFound := serverStateA2S[server.ServerNameShort]
			if a2sFound {
				if a2sInfo.Name != "" && state.Name != a2sInfo.Name {
					state.Name = a2sInfo.Name
				}
				state.NameA2S = a2sInfo.Name
				state.Protocol = a2sInfo.Protocol
				state.Map = a2sInfo.Map
				state.Folder = a2sInfo.Folder
				state.Game = a2sInfo.Game
				state.AppId = a2sInfo.ID
				state.PlayerCount = int(a2sInfo.Players)
				state.MaxPlayers = int(a2sInfo.MaxPlayers)
				state.Bots = int(a2sInfo.Bots)
				state.ServerType = a2sInfo.ServerType.String()
				state.ServerOS = a2sInfo.ServerOS.String()
				state.Password = !a2sInfo.Visibility
				state.VAC = a2sInfo.VAC
				state.Version = a2sInfo.Version
				if a2sInfo.SourceTV != nil {
					state.STVPort = a2sInfo.SourceTV.Port
					state.STVName = a2sInfo.SourceTV.Name
				}
				if a2sInfo.ExtendedServerInfo != nil {
					state.SteamID = steamid.SID64(a2sInfo.ExtendedServerInfo.SteamID)
					state.GameID = a2sInfo.ExtendedServerInfo.GameID
					state.Keywords = strings.Split(a2sInfo.ExtendedServerInfo.Keywords, ",")
				}
			}
			statusInfo, statusFound := serverStateStatus[server.ServerNameShort]
			if statusFound {
				if state.Name != statusInfo.ServerName && statusInfo.ServerName == "" {
					state.Name = statusInfo.ServerName
				}

				var knownPlayers []model.ServerStatePlayer
				for _, player := range statusInfo.Players {
					var newPlayer model.ServerStatePlayer
					newPlayer.UserID = player.UserID
					newPlayer.Name = player.Name
					newPlayer.SID = player.SID
					newPlayer.ConnectedTime = player.ConnectedTime
					newPlayer.State = player.State
					newPlayer.Ping = player.Ping
					newPlayer.Loss = player.Loss
					newPlayer.IP = player.IP
					newPlayer.Port = player.Port
					knownPlayers = append(knownPlayers, newPlayer)
				}
				state.Players = knownPlayers
			}
			newState = append(newState, state)
		}
		serverStateMu.Lock()
		serverState = newState
		serverStateMu.Unlock()
		return nil
	}
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if errUpdate := refreshState(); errUpdate != nil {
				log.Errorf("Failed to refreshState server state: %v", errUpdate)
			}
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
func mapChanger(ctx context.Context, database store.ServerStore, timeout time.Duration) {
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
			stateCopy := ServerState()
			for _, state := range stateCopy {
				activity, activityFound := activityMap[state.NameShort]
				if !activityFound || len(state.Players) > 0 {
					activityMap[state.NameShort] = &at{config.Now(), false}
					continue
				}
				if !activity.triggered && time.Since(activity.lastActive) > timeout {
					isDefaultMap := false
					for _, m := range config.General.DefaultMaps {
						if m == state.Map {
							isDefaultMap = true
							break
						}
					}
					if isDefaultMap {
						continue
					}
					var server model.Server
					if errGetServer := database.GetServerByName(ctx, state.NameShort, &server); errGetServer != nil {
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
					if server.DefaultMap == state.Map {
						continue
					}
					go func(s model.Server, mapName string) {
						var logger = log.WithFields(log.Fields{"map": nextMap, "reason": "no_activity", "server": state.ServerId})
						logger.Infof("Idle map change triggered")
						if _, errExecRCON := query.ExecRCON(ctx, server, fmt.Sprintf("changelevel %s", mapName)); errExecRCON != nil {
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
func banSweeper(ctx context.Context, database store.Store) {
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
						if errDrop := database.DropBan(ctx, &expiredBan, false); errDrop != nil {
							log.Errorf("Failed to drop expired expiredBan: %v", errDrop)
						} else {
							banType := "Ban"
							if expiredBan.BanType == model.NoComm {
								banType = "Mute"
							}
							var person model.Person
							if errPerson := database.GetOrCreatePersonBySteamID(ctx, expiredBan.SteamID, &person); errPerson != nil {
								log.Errorf("Failed to get expired person: %v", errPerson)
								continue
							}
							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}
							log.WithFields(log.Fields{
								"sid":    expiredBan.SteamID,
								"name":   name,
								"origin": expiredBan.Source.String(),
								"reason": expiredBan.Reason.String(),
								"custom": expiredBan.ReasonText,
							}).Infof("%s expired", banType)
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
							log.Infof("CIDR ban expired: %v", expiredNetBan)
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
