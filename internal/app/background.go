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
	"sync"
	"time"
)

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func (g *gbans) profileUpdater() {
	var update = func() {
		ctx, cancel := context.WithTimeout(g.ctx, time.Second*10)
		defer cancel()
		people, pErr := g.db.GetExpiredProfiles(ctx, 100)
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
			if err4 := g.db.GetOrCreatePersonBySteamID(g.ctx, sid, &p); err4 != nil {
				log.Errorf("Failed to get person: %v", err4)
				continue
			}
			p.PlayerSummary = &s
			if err5 := g.db.SavePerson(g.ctx, &p); err5 != nil {
				log.Errorf("Failed to save person: %v", err5)
				continue
			}
		}
		log.Debugf("Updated %d profiles", len(summaries))
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-g.ctx.Done():
			log.Debugf("profileUpdater shutting down")
			return
		}
	}

}

// serverStateUpdater concurrently ( num_servers * 2) updates all known servers' A2S and rcon status
// information. This data is accessed often so it is cached
func (g *gbans) serverStateUpdater() {
	freq, errD := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errD != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errD)
	}
	var update = func(ctx context.Context) {
		servers, err := g.db.GetServers(ctx, false)
		if err != nil {
			log.Errorf("Failed to fetch servers to update: %v", err)
			return
		}
		newServers := map[string]model.ServerState{}
		newServersMu := &sync.RWMutex{}
		wg := &sync.WaitGroup{}
		for _, srv := range servers {
			ss := model.ServerState{}
			ss.Region = srv.Region
			ss.Enabled = srv.IsEnabled
			ss.CountryCode = srv.CC
			ss.Name = srv.ServerName
			ss.Reserved = 8
			wg.Add(1)
			go func(server model.Server) {
				defer wg.Done()
				iwg := &sync.WaitGroup{}
				iwg.Add(2)
				go func() {
					defer iwg.Done()
					status, errS := query.GetServerStatus(server)
					if errS != nil {
						log.Debugf("Failed to update server status: %v", errS)
						return
					}
					ss.Status = status
				}()
				go func() {
					defer iwg.Done()
					a, errA := query.A2SQueryServer(server)
					if errA != nil {
						log.Debugf("Failed to update a2s status: %v", errA)
						return
					}
					ss.A2S = *a
				}()
				iwg.Wait()
				ss.LastUpdate = time.Now()
				newServersMu.Lock()
				newServers[server.ServerName] = ss
				newServersMu.Unlock()
			}(srv)
		}
		wg.Wait()
		g.serversStateMu.Lock()
		g.serversState = newServers
		g.serversStateMu.Unlock()
		log.Debugf("Updated %d servers", len(servers))
	}
	update(g.ctx)
	ticker := time.NewTicker(freq)
	for {
		select {
		case <-ticker.C:
			update(g.ctx)
		case <-g.ctx.Done():
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
func (g *gbans) mapChanger(timeout time.Duration) {
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
			g.serversStateMu.RLock()
			stateCopy := g.ServerState()
			g.serversStateMu.RUnlock()
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
					if err := g.db.GetServerByName(context.Background(), serverId, &srv); err != nil {
						g.l.Errorf("Failed to get server for map changer: %v", err)
						continue
					}
					nextMap := srv.DefaultMap
					if nextMap == "" {
						nextMap = config.General.DefaultMaps[rand.Intn(len(config.General.DefaultMaps))]
					}
					if nextMap == "" {
						g.l.Errorf("Failed to get valid nextMap value")
						continue
					}
					if srv.DefaultMap == state.Status.Map {
						continue
					}
					go func(s model.Server, mapName string) {
						var l = g.l.WithFields(log.Fields{"map": nextMap, "reason": "no_activity", "srv": serverId})
						l.Infof("Idle map change triggered")
						if _, err := query.ExecRCON(srv, fmt.Sprintf("changelevel %s", mapName)); err != nil {
							g.l.Errorf("failed to exec mapchanger rcon: %v", err)
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
		case <-g.ctx.Done():
			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func (g *gbans) banSweeper() {
	log.Debug("ban sweeper routine started")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			bans, err := g.db.GetExpiredBans(g.ctx)
			if err != nil && !errors.Is(err, store.ErrNoResult) {
				log.Warnf("Failed to get expired bans: %v", err)
			} else {
				for _, ban := range bans {
					if err := g.db.DropBan(g.ctx, &ban); err != nil {
						log.Errorf("Failed to drop expired ban: %v", err)
					} else {
						log.Infof("ban expired: %v", ban)
					}
				}
			}
			netBans, err2 := g.db.GetExpiredNetBans(g.ctx)
			if err2 != nil && !errors.Is(err2, store.ErrNoResult) {
				log.Warnf("Failed to get expired netbans: %v", err2)
			} else {
				for _, ban := range netBans {
					if err := g.db.DropBanNet(g.ctx, &ban); err != nil {
						log.Errorf("Failed to drop expired network ban: %v", err)
					} else {
						log.Infof("Network ban expired: %v", ban)
					}
				}
			}
		case <-g.ctx.Done():
			log.Debugf("banSweeper shutting down")
			return
		}
	}
}
