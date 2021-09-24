package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

// Find will attempt to match an input string to a steam id and if connected, a
// matching active Player.
//
// Will accept SteamID or partial Player names. When using a partial Player name, the
// first instance that contains the partial match will be returned.
//
// Valid will be set to true if the value is a Valid steamid, even if the Player is not
// actively connected
func (g Gbans) Find(playerStr string, ip string, pi *model.PlayerInfo) error {
	var (
		result = &model.PlayerInfo{
			Player: &extra.Player{},
			Server: &model.Server{},
		}
		err      error
		inGame   = false
		valid    = false
		foundSid steamid.SID64
	)
	if ip != "" {
		err = g.findPlayerByIP(g.ctx, net.ParseIP(ip), pi)
		if err == nil {
			foundSid = result.Player.SID
			inGame = true
		}
	} else {
		if err = g.findPlayerByName(g.ctx, playerStr, pi); err == nil {
			foundSid = result.Player.SID
			inGame = true
		}
	}
	if pi != nil && pi.Player != nil && pi.Player.SID.Valid() || foundSid.Valid() {
		valid = true
	}
	pi.Valid = valid
	pi.InGame = inGame
	return nil
}

func (g *Gbans) findPlayerByName(ctx context.Context, name string, pi *model.PlayerInfo) error {
	name = strings.ToLower(name)
	for serverId, status := range g.ServerState() {
		for _, player := range status.Status.Players {
			if strings.Contains(strings.ToLower(player.Name), name) {
				var srv model.Server
				if errGS := g.db.GetServerByName(ctx, serverId, &srv); errGS != nil {
					return errGS
				}
				pi.Valid = true
				pi.InGame = true
				pi.Server = &srv
				pi.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

func (g *Gbans) findPlayerBySID(ctx context.Context, sid steamid.SID64, pi *model.PlayerInfo) error {
	for serverId, status := range g.ServerState() {
		for _, player := range status.Status.Players {
			if player.SID == sid {
				var srv model.Server
				if errGS := g.db.GetServerByName(ctx, serverId, &srv); errGS != nil {
					return errGS
				}
				pi.Valid = true
				pi.InGame = true
				pi.Server = &srv
				pi.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

func (g *Gbans) findPlayerByIP(ctx context.Context, ip net.IP, pi *model.PlayerInfo) error {
	for serverId, status := range g.ServerState() {
		for _, player := range status.Players {
			if ip.Equal(player.IP) {
				var srv model.Server
				if errGS := g.db.GetServerByName(ctx, serverId, &srv); errGS != nil {
					return errGS
				}
				pi.Valid = true
				pi.InGame = true
				pi.Server = &srv
				pi.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

// ServerState returns a copy of the current known state for all servers.
func (g *Gbans) ServerState() model.ServerStateCollection {
	g.serversStateMu.RLock()
	state := g.serversState
	g.serversStateMu.RUnlock()
	return state
}

// FindPlayerByCIDR  looks for a player with a ip intersecting with the cidr range
// TODO Support matching multiple people and not just the first found
func (g Gbans) FindPlayerByCIDR(ipNet *net.IPNet, pi *model.PlayerInfo) error {
	for serverId, status := range g.ServerState() {
		for _, player := range status.Players {
			if ipNet.Contains(player.IP) {
				c, cancel := context.WithTimeout(g.ctx, time.Second*5)
				var srv model.Server
				if errGS := g.db.GetServerByName(c, serverId, &srv); errGS != nil {
					cancel()
					return errGS
				}
				pi.Valid = true
				pi.InGame = true
				pi.Server = &srv
				pi.Player = &player
				cancel()
			}
		}
	}
	return consts.ErrUnknownID
}

// GetOrCreateProfileBySteamID functions the same as GetOrCreatePersonBySteamID except
// that it will also query the steam webapi to fetch and load the extra Player summary info
func (g Gbans) GetOrCreateProfileBySteamID(ctx context.Context, sid steamid.SID64, ipAddr string, p *model.Person) error {
	sum, err := steamweb.PlayerSummaries(steamid.Collection{sid})
	if err != nil {
		return errors.Wrapf(err, "Failed to get Player summary: %v", err)
	}
	vac, errBans := steam.FetchPlayerBans(steamid.Collection{sid})
	if errBans != nil || len(vac) != 1 {
		return errors.Wrapf(err, "Failed to get Player ban state: %v", err)
	}
	if errGP := g.db.GetOrCreatePersonBySteamID(ctx, sid, p); err != nil {
		return errors.Wrapf(errGP, "Failed to get person: %d", sid)
	}
	p.SteamID = sid
	p.CommunityBanned = vac[0].CommunityBanned
	p.VACBans = vac[0].NumberOfVACBans
	p.GameBans = vac[0].NumberOfGameBans
	p.EconomyBan = vac[0].EconomyBan
	p.CommunityBanned = vac[0].CommunityBanned
	p.DaysSinceLastBan = vac[0].DaysSinceLastBan
	if len(sum) > 0 {
		s := sum[0]
		p.PlayerSummary = &s
	} else {
		log.Warnf("Failed to fetch Player summary for: %v", sid)
	}
	if errSave := g.db.SavePerson(ctx, p); errSave != nil {
		return errors.Wrapf(errSave, "Failed to save person")
	}
	if ipAddr != "" && !p.IPAddr.Equal(net.ParseIP(ipAddr)) {
		if errIP := g.db.AddPersonIP(ctx, p, ipAddr); errIP != nil {
			return errors.Wrapf(errIP, "Could not add ip record")
		}
		p.IPAddr = net.ParseIP(ipAddr)
	}
	log.Debugf("Profile updated successfully: %s [%d]", p.PersonaName, p.SteamID.Int64())
	return nil
}
