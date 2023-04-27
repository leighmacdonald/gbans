package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"net"
	"sort"
	"strings"
	"time"
)

func (app *App) ServerState() state.ServerStateCollection {
	app.serverStateMu.RLock()
	s := app.serverState
	app.serverStateMu.RUnlock()
	sort.Slice(s, func(i, j int) bool {
		return s[i].NameShort < s[j].NameShort
	})
	return s
}

// Find will attempt to match an input string to a steam id and if connected, a
// matching active Player.
//
// Will accept SteamID or partial Player names. When using a partial Player name, the
// first instance that contains the partial match will be returned.
//
// Valid will be set to true if the value is a Valid steamid, even if the Player is not
// actively connected
//
// TODO cleanup this mess
func (app *App) Find(ctx context.Context, playerStr store.StringSID, ip string, playerInfo *state.PlayerInfo) error {
	var (
		result = &state.PlayerInfo{
			Player: &state.ServerStatePlayer{},
			Server: &store.Server{},
		}
		err      error
		inGame   = false
		valid    = false
		foundSid steamid.SID64
	)
	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if ip != "" {
		err = app.findPlayerByIP(c, net.ParseIP(ip), playerInfo)
		if err == nil {
			foundSid = result.Player.SID
			inGame = true
		}
	} else {
		found := false
		sid, errSid := steamid.StringToSID64(string(playerStr))
		if errSid == nil && sid.Valid() {
			if errPSID := app.findPlayerBySID(c, sid, playerInfo); errPSID == nil {
				foundSid = sid
				found = true
				inGame = true
			}
		}
		if !found {
			if err = app.findPlayerByName(c, string(playerStr), playerInfo); err == nil {
				foundSid = result.Player.SID
				inGame = true
			}
		}
	}
	if playerInfo != nil && playerInfo.Player != nil && playerInfo.Player.SID.Valid() || foundSid.Valid() {
		playerInfo.SteamID = playerInfo.Player.SID
		valid = true
	}
	playerInfo.Valid = valid
	playerInfo.InGame = inGame
	return nil
}

func (app *App) findPlayerByName(ctx context.Context, name string, playerInfo *state.PlayerInfo) error {
	for _, currentState := range app.ServerState() {
		for _, player := range currentState.Players {
			if strings.Contains(strings.ToLower(player.Name), strings.ToLower(name)) {
				var server store.Server
				if errGetServerByName := app.store.GetServerByName(ctx, currentState.NameShort, &server); errGetServerByName != nil {
					return errGetServerByName
				}
				playerInfo.Valid = true
				playerInfo.InGame = true
				playerInfo.Server = &server
				playerInfo.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

func (app *App) findPlayerBySID(ctx context.Context, sid steamid.SID64, playerInfo *state.PlayerInfo) error {
	for _, currentState := range app.ServerState() {
		for _, player := range currentState.Players {
			if player.SID == sid {
				var server store.Server
				if errGetServer := app.store.GetServerByName(ctx, currentState.NameShort, &server); errGetServer != nil {
					return errGetServer
				}
				playerInfo.Valid = true
				playerInfo.InGame = true
				playerInfo.Server = &server
				playerInfo.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

func (app *App) findPlayerByIP(ctx context.Context, ip net.IP, playerInfo *state.PlayerInfo) error {
	for _, currentState := range app.ServerState() {
		for _, player := range currentState.Players {
			if ip.Equal(player.IP) {
				var server store.Server
				if errGetServer := app.store.GetServerByName(ctx, currentState.NameShort, &server); errGetServer != nil {
					return errGetServer
				}
				playerInfo.Valid = true
				playerInfo.InGame = true
				playerInfo.Server = &server
				playerInfo.Player = &player
				return nil
			}
		}
	}
	return consts.ErrUnknownID
}

// FindPlayerByCIDR  looks for a player with a ip intersecting with the cidr range
// TODO Support matching multiple people and not just the first found
func (app *App) FindPlayerByCIDR(ctx context.Context, ipNet *net.IPNet, playerInfo *state.PlayerInfo) error {
	for _, currentState := range app.ServerState() {
		for _, player := range currentState.Players {
			if ipNet.Contains(player.IP) {
				localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				var server store.Server
				if errGetServer := app.store.GetServerByName(localCtx, currentState.NameShort, &server); errGetServer != nil {
					cancel()
					return errGetServer
				}
				playerInfo.Valid = true
				playerInfo.InGame = true
				playerInfo.Server = &server
				playerInfo.Player = &player
				cancel()
			}
		}
	}
	return consts.ErrUnknownID
}

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date
func (app *App) PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error {
	if errGetPerson := app.store.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get person instance: %d", sid)
	}
	if person.IsNew || config.Now().Sub(person.UpdatedOnSteam) > time.Minute*60 {
		summaries, errSummaries := steamweb.PlayerSummaries(steamid.Collection{sid})
		if errSummaries != nil {
			return errors.Wrapf(errSummaries, "Failed to get Player summary: %v", errSummaries)
		}
		if len(summaries) > 0 {
			s := summaries[0]
			person.PlayerSummary = &s
		} else {
			return errors.Errorf("Failed to fetch Player summary for %d", sid)
		}
		vac, errBans := thirdparty.FetchPlayerBans(steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			return errors.Wrapf(errSummaries, "Failed to get Player ban state: %v", errSummaries)
		} else {
			person.CommunityBanned = vac[0].CommunityBanned
			person.VACBans = vac[0].NumberOfVACBans
			person.GameBans = vac[0].NumberOfGameBans
			person.EconomyBan = vac[0].EconomyBan
			person.CommunityBanned = vac[0].CommunityBanned
			person.DaysSinceLastBan = vac[0].DaysSinceLastBan
		}
		person.UpdatedOnSteam = config.Now()
	}
	person.SteamID = sid
	if errSavePerson := app.store.SavePerson(ctx, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}
	return nil
}
