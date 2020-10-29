package service

import (
	"context"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func profileUpdater() {
	var update = func() {
		o := store.NewQueryOpts()
		o.Limit = 5 // Max per query of WebAPI
		loop := 0
		for {
			o.Offset = loop * o.Limit
			bans, err := store.GetBansOlderThan(o, time.Now().Add(-(time.Hour * 24)))
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
			summaries, err := extra.PlayerSummaries(context.Background(), sids)
			cnt := 0
			for _, s := range summaries {
				sid, err := steamid.SID64FromString(s.Steamid)
				if err != nil {
					log.Errorf("Failed to parse steamid from webapi: %v", err)
					continue
				}
				p, err := store.GetOrCreatePersonBySteamID(sid)
				if err != nil {
					log.Errorf("Failed to get person: %v", err)
					continue
				}
				p.PlayerSummary = s
				if err := store.SavePerson(&p); err != nil {
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
			return
		}
	}

}

func serverStateUpdater() {
	var update = func() {
		servers, err := store.GetServers()
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
			respRCON = QueryRCON(context.Background(), servers, "status")
		}()
		go func() {
			defer wg.Done()
			respA2S = QueryA2SInfo(context.Background(), servers)
		}()
		wg.Wait()
		for name, resp := range respRCON {
			s, err := extra.ParseStatus(resp, true)
			if err != nil {
				log.Warnf("Failed to parse server state (%s): %v", name, err)
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
			serverStateMu.Lock()
			serverState[name] = ServerState{addr, port, slots, tf2, a2sinfo, s}
			serverStateMu.Unlock()
		}
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			return
		}
	}
}

func banSweeper() {
	log.Debug("Ban sweeper routine started")
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			bans, err := store.GetExpiredBans()
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range bans {
					if err := store.DropBan(ban); err != nil {
						log.Errorf("Failed to drop expired ban: %v", err)
					} else {
						log.Infof("Ban expired: %v", ban)
					}
				}
			}
			netBans, err := store.GetExpiredNetBans()
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range netBans {
					if err := store.DropNetBan(ban); err != nil {
						log.Errorf("Failed to drop expired network ban: %v", err)
					} else {
						log.Infof("Network ban expired: %v", ban)
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func updateSearchIndex() {
	if err := store.UpdateIndex(); err != nil {
		log.Errorf("Failed to update search index")
	}
	t := time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-t.C:
			if err := store.UpdateIndex(); err != nil {
				log.Errorf("Failed to update search index")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
