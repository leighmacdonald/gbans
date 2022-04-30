package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/store"
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
//
// TODO cleanup this mess
func Find(ctx context.Context, database store.Store, playerStr model.Target, ip string, playerInfo *model.PlayerInfo) error {
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
	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if ip != "" {
		err = findPlayerByIP(c, database, net.ParseIP(ip), playerInfo)
		if err == nil {
			foundSid = result.Player.SID
			inGame = true
		}
	} else {
		found := false
		sid, errSid := steamid.StringToSID64(string(playerStr))
		if errSid == nil && sid.Valid() {
			if errPSID := findPlayerBySID(c, database, sid, playerInfo); errPSID == nil {
				foundSid = sid
				found = true
				inGame = true
			}
		}
		if !found {
			if err = findPlayerByName(c, database, string(playerStr), playerInfo); err == nil {
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

func findPlayerByName(ctx context.Context, database store.ServerStore, name string, playerInfo *model.PlayerInfo) error {
	for serverId, serverState := range ServerState() {
		for _, player := range serverState.Status.Players {
			if strings.Contains(strings.ToLower(player.Name), strings.ToLower(name)) {
				var server model.Server
				if errGetServerByName := database.GetServerByName(ctx, serverId, &server); errGetServerByName != nil {
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

func findPlayerBySID(ctx context.Context, database store.ServerStore, sid steamid.SID64, playerInfo *model.PlayerInfo) error {
	for serverId, serverState := range ServerState() {
		for _, player := range serverState.Status.Players {
			if player.SID == sid {
				var server model.Server
				if errGetServer := database.GetServerByName(ctx, serverId, &server); errGetServer != nil {
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

func findPlayerByIP(ctx context.Context, database store.ServerStore, ip net.IP, playerInfo *model.PlayerInfo) error {
	for serverId, serverState := range ServerState() {
		for _, player := range serverState.Players {
			if ip.Equal(player.IP) {
				var server model.Server
				if errGetServer := database.GetServerByName(ctx, serverId, &server); errGetServer != nil {
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

// ServerState returns a copy of the current known state for all servers.
func ServerState() model.ServerStateCollection {
	roState := model.ServerStateCollection{}
	serversStateMu.RLock()
	defer serversStateMu.RUnlock()
	for serverId, serverState := range serversState {
		roState[serverId] = serverState
	}
	return roState
}

// FindPlayerByCIDR  looks for a player with a ip intersecting with the cidr range
// TODO Support matching multiple people and not just the first found
func FindPlayerByCIDR(ctx context.Context, database store.ServerStore, ipNet *net.IPNet, playerInfo *model.PlayerInfo) error {
	for serverId, serverState := range ServerState() {
		for _, player := range serverState.Players {
			if ipNet.Contains(player.IP) {
				localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				var server model.Server
				if errGetServer := database.GetServerByName(localCtx, serverId, &server); errGetServer != nil {
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

// getOrCreateProfileBySteamID functions the same as GetOrCreatePersonBySteamID except
// that it will also query the steam webapi to fetch and load the extra Player summary info
func getOrCreateProfileBySteamID(ctx context.Context, database store.PersonStore, sid steamid.SID64, ipAddr string, person *model.Person) error {
	if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
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
		vac, errBans := steam.FetchPlayerBans(steamid.Collection{sid})
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
		log.WithFields(log.Fields{"age": config.Now().Sub(person.UpdatedOnSteam).String()}).
			Debugf("Profile updated successfully: %s [%d]", person.PersonaName, person.SteamID.Int64())
	}
	person.SteamID = sid
	if errSavePerson := database.SavePerson(ctx, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}
	return nil
}
