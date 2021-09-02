package app

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func (g *Gbans) profileUpdater() {
	var update = func() {
		o := store.NewQueryFilter("")
		o.Limit = 5 // Max per query of WebAPI
		loop := uint64(0)
		for {
			o.Offset = loop * o.Limit
			bans, err := g.db.GetBansOlderThan(g.ctx, o, config.Now().Add(-(time.Hour * 24)))
			if err != nil {
				log.Warnf("Failed to get old bans for update: %v", err)
				break
			}
			if bans == nil {
				break
			}
			var sids steamid.Collection
			for _, b := range bans {
				sids = append(sids, b.SteamID)
			}
			summaries, err2 := steamweb.PlayerSummaries(sids)
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
				var p model.Person
				if err4 := g.db.GetOrCreatePersonBySteamID(g.ctx, sid, &p); err4 != nil {
					log.Errorf("Failed to get person: %v", err4)
					continue
				}
				p.PlayerSummary = &s
				if err5 := g.db.SavePerson(g.ctx, &p); err5 != nil {
					log.Errorf("Failed to save person: %v", err5)
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
		case <-g.ctx.Done():
			log.Debugf("profileUpdater shutting down")
			return
		}
	}

}

func (g *Gbans) serverStateUpdater() {
	var update = func() {
		servers, err := g.db.GetServers(g.ctx)
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
			respRCON = query.RCON(g.ctx, servers, "status")
		}()
		go func() {
			defer wg.Done()
			respA2S = query.A2SInfo(servers)
		}()
		wg.Wait()
		for name, resp := range respRCON {
			s, errPs := extra.ParseStatus(resp, true)
			if errPs != nil {
				log.Warnf("Failed to parse server state (%s): %v", name, errPs)
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
				log.Warnf("Failed to get a2s server info for: %s", name)
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
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Gbans) banSweeper() {
	log.Debug("ban sweeper routine started")
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			bans, err := g.db.GetExpiredBans(g.ctx)
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range bans {
					if err := g.db.DropBan(g.ctx, ban); err != nil {
						log.Errorf("Failed to drop expired ban: %v", err)
					} else {
						log.Infof("ban expired: %v", ban)
					}
				}
			}
			netBans, err2 := g.db.GetExpiredNetBans(g.ctx)
			if err2 != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range netBans {
					if err := g.db.DropNetBan(g.ctx, ban); err != nil {
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
