package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func profileUpdater(ctx context.Context) {
	var update = func() {
		o := store.NewQueryFilter("")
		o.Limit = 5 // Max per query of WebAPI
		loop := uint64(0)
		for {
			o.Offset = loop * o.Limit
			bans, err := store.GetBansOlderThan(ctx, o, config.Now().Add(-(time.Hour * 24)))
			if err != nil {
				log.Warnf("Failed to get old bans for update: %v", err)
				break
			}
			if bans == nil {
				break
			}
			var sids []steamid.SID64
			for _, b := range bans {
				sids = append(sids, b.SteamID)
			}
			summaries, err2 := extra.PlayerSummaries(context.Background(), sids)
			if err2 != nil {
				log.Errorf("Failed to get Player summaries: %v", err2)
				continue
			}
			cnt := 0
			for _, s := range summaries {
				sid, err3 := steamid.SID64FromString(s.Steamid)
				if err3 != nil {
					log.Errorf("Failed to parse steamid from webapi: %v", err3)
					continue
				}
				p, err4 := store.GetOrCreatePersonBySteamID(ctx, sid)
				if err4 != nil {
					log.Errorf("Failed to get person: %v", err4)
					continue
				}
				p.PlayerSummary = &s
				if err := store.SavePerson(ctx, p); err != nil {
					log.Errorf("Failed to save person: %v", err)
					continue
				}
				cnt++
			}
			log.Debugf("Updated %d profiles", cnt)
			loop++
		}
	}
	update()
	ticker := time.NewTicker(time.Hour)
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

func serverStateUpdater(ctx context.Context) {
	var update = func() {
		servers, err := store.GetServers(ctx)
		if err != nil {
			log.Errorf("Failed to fetch servers to update")
			return
		}
		wg := &sync.WaitGroup{}
		wg.Add(2)
		respRCON := map[string]string{}
		respA2S := map[string]*a2s.ServerInfo{}
		go func() {
			defer wg.Done()
			respRCON = query.RCON(ctx, servers, "status")
		}()
		go func() {
			defer wg.Done()
			respA2S = query.A2SInfo(servers)
		}()
		wg.Wait()
		for name, resp := range respRCON {
			s, errPs := extra.ParseStatus(resp, true)
			if errPs != nil {
				log.Warnf("Failed to parse IsServer state (%s): %v", name, errPs)
				return
			}
			var (
				addr  string
				port  int
				slots int
			)
			for _, srv := range servers {
				if srv.ServerName == name {
					addr = srv.Address
					port = srv.Port
					slots = srv.Slots(s.PlayersMax)
					break
				}
			}
			a2sinfo, found := respA2S[name]
			if !found {
				log.Warnf("Failed to get a2s IsServer info for: %s", name)
			}
			state.SetServer(name, state.ServerState{
				Addr: addr, Port: port, Slots: slots, GameType: state.TF2, A2SInfo: a2sinfo, Status: s, Alive: found})
		}
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-gCtx.Done():
			return
		}
	}
}

func banSweeper(ctx context.Context) {
	log.Debug("ban sweeper routine started")
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			bans, err := store.GetExpiredBans(ctx)
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range bans {
					if err := store.DropBan(ctx, ban); err != nil {
						log.Errorf("Failed to drop expired ban: %v", err)
					} else {
						log.Infof("ban expired: %v", ban)
					}
				}
			}
			netBans, err2 := store.GetExpiredNetBans(ctx)
			if err2 != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range netBans {
					if err := store.DropNetBan(ctx, ban); err != nil {
						log.Errorf("Failed to drop expired network ban: %v", err)
					} else {
						log.Infof("Network ban expired: %v", ban)
					}
				}
			}
		case <-ctx.Done():
			log.Debugf("banSweeper shutting down")
			return
		}
	}
}
