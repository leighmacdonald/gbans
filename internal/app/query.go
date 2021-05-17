package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
)

// FindPlayer will attempt to match a input string to a steam id and if connected, a
// matching active Player.
//
// Will accept SteamID or partial Player names. When using a partial Player name, the
// first instance that contains the partial match will be returned.
//
// Valid will be set to true if the value is a Valid steamid, even if the Player is not
// actively connected
func (a ActionHandlers) FindPlayer(ctx context.Context, playerStr string, ip string) model.PlayerInfo {
	var (
		player   *extra.Player
		server   *model.Server
		err      error
		sid      steamid.SID64
		inGame   = false
		foundSid steamid.SID64
		valid    = false
	)
	if ip != "" {
		player, server, err = findPlayerByIP(ctx, net.ParseIP(ip))
		if err == nil {
			foundSid = player.SID
			inGame = true
		}
	} else {
		sidFS, errFS := steamid.ResolveSID64(ctx, playerStr)
		if errFS == nil && sidFS.Valid() {
			foundSid = sidFS
			player, server, err = findPlayerBySID(ctx, sidFS)
			if err == nil {
				inGame = true
			}
		} else {
			player, server, err = findPlayerByName(ctx, playerStr)
			if err == nil {
				foundSid = player.SID
				inGame = true
			}
		}
	}
	if sid.Valid() || foundSid.Valid() {
		valid = true
	}
	return model.PlayerInfo{Player: player, Server: server, SteamID: foundSid, InGame: inGame, Valid: valid}
}

func findPlayerByName(ctx context.Context, name string) (*extra.Player, *model.Server, error) {
	name = strings.ToLower(name)
	statuses, err := getAllServerStatus(ctx)
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if strings.Contains(strings.ToLower(player.Name), name) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, consts.ErrUnknownID
}

func findPlayerBySID(ctx context.Context, sid steamid.SID64) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus(ctx)
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if player.SID == sid {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, consts.ErrUnknownID
}

func findPlayerByIP(ctx context.Context, ip net.IP) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus(ctx)
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if ip.Equal(player.IP) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, consts.ErrUnknownID
}

func getAllServerStatus(ctx context.Context) (map[model.Server]extra.Status, error) {
	servers, err := store.GetServers(ctx)
	if err != nil {
		return nil, err
	}
	statuses := make(map[model.Server]extra.Status)
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			status, err2 := query.GetServerStatus(server)
			if err2 != nil {
				log.Errorf("Failed to parse status output: %v", err2)
				return
			}
			mu.Lock()
			statuses[server] = status
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return statuses, nil
}

func (a ActionHandlers) FindPlayerByCIDR(ctx context.Context, ipNet *net.IPNet) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus(ctx)
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if ipNet.Contains(player.IP) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, consts.ErrUnknownID
}

// GetOrCreateProfileBySteamID functions the same as GetOrCreatePersonBySteamID except
// that it will also query the steam webapi to fetch and load the extra Player summary info
func (a ActionHandlers) GetOrCreateProfileBySteamID(ctx context.Context, sid steamid.SID64, ipAddr string) (*model.Person, error) {
	// TODO make these non-fatal errors?
	sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Player summary: %v", err)
	}
	vac, errBans := steam.FetchPlayerBans(ctx, []steamid.SID64{sid})
	if errBans != nil || len(vac) != 1 {
		return nil, errors.Wrapf(err, "Failed to get Player ban state: %v", err)
	}
	p, err := store.GetOrCreatePersonBySteamID(ctx, sid)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get person: %v", err)
	}
	p.SteamID = sid
	p.CommunityBanned = vac[0].CommunityBanned
	p.VACBans = vac[0].VACBans
	p.GameBans = vac[0].GameBans
	p.EconomyBan = vac[0].EconomyBan
	p.CommunityBanned = vac[0].CommunityBanned
	p.DaysSinceLastBan = vac[0].DaysSinceLastBan
	if len(sum) > 0 {
		s := sum[0]
		p.PlayerSummary = &s
	} else {
		log.Warnf("Failed to fetch Player summary for: %v", sid)
	}
	if errSave := store.SavePerson(ctx, p); errSave != nil {
		return nil, errors.Wrapf(errSave, "Failed to save person")
	}
	if ipAddr != "" && !p.IPAddr.Equal(net.ParseIP(ipAddr)) {
		if errIP := store.AddPersonIP(ctx, p, ipAddr); errIP != nil {
			return nil, errors.Wrapf(errIP, "Could not add ip record")
		}
		p.IPAddr = net.ParseIP(ipAddr)
	}
	log.Debugf("Profile updated successfully: %s [%d]", p.PersonaName, p.SteamID.Int64())
	return p, nil
}
