package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
	"time"
)

func (app *App) sendDiscordPayload(payload discordPayload) {
	log.WithFields(log.Fields{
		"channel": payload.channelId,
		"enabled": config.Discord.PublicLogChannelEnable,
	}).Tracef("Sending discord payload")
	if config.Discord.PublicLogChannelEnable {
		select {
		case app.discordSendMsg <- payload:
		default:
			log.Warnf("Cannot send discord payload, channel full")
		}
	}
}

// Kick will kick the steam id from whatever server it is connected to.
func (app *App) Kick(ctx context.Context, database store.Store, origin model.Origin, target model.StringSID, author model.StringSID,
	reason model.Reason, playerInfo *model.PlayerInfo) error {
	authorSid64, errAid := author.SID64()
	if errAid != nil {
		return errAid
	}
	// kick the user if they currently are playing on a server
	var foundPI model.PlayerInfo
	if errFind := app.Find(ctx, target, "", &foundPI); errFind != nil {
		return errFind
	}
	if foundPI.Valid && foundPI.InGame {
		rconResponse, errExecRCON := query.ExecRCON(ctx, *foundPI.Server, fmt.Sprintf("sm_kick #%d %s", foundPI.Player.UserID, reason))
		if errExecRCON != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errExecRCON)
			return errExecRCON
		}
		log.Debugf("RCON response: %s", rconResponse)
		log.WithFields(log.Fields{"origin": origin, "target": target, "author": util.SanitizeLog(authorSid64.String())}).
			Infof("User kicked")
	}
	if playerInfo != nil {
		*playerInfo = foundPI
	}

	return nil
}

// Silence will gag & mute a player
func (app *App) Silence(ctx context.Context, origin model.Origin, target model.StringSID, author model.StringSID,
	reason model.Reason, playerInfo *model.PlayerInfo) error {
	authorSid64, errAid := author.SID64()
	if errAid != nil {
		return errAid
	}
	// kick the user if they currently are playing on a server
	var foundPI model.PlayerInfo
	if errFind := app.Find(ctx, target, "", &foundPI); errFind != nil {
		return errFind
	}
	if foundPI.Valid && foundPI.InGame {
		rconResponse, errExecRCON := query.ExecRCON(
			ctx,
			*foundPI.Server,
			fmt.Sprintf(`sm_silence "#%s" %s`, steamid.SID64ToSID(foundPI.Player.SID), reason),
		)
		if errExecRCON != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errExecRCON)
			return errExecRCON
		}
		log.Debugf("RCON response: %s", rconResponse)
		log.WithFields(log.Fields{
			"origin": origin,
			"target": target,
			"author": util.SanitizeLog(authorSid64.String())}).Infof("User silenced")
	}
	if playerInfo != nil {
		*playerInfo = foundPI
	}

	return nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the bot does not require more privileged intents.
func (app *App) SetSteam(ctx context.Context, sid64 steamid.SID64, discordId string) error {
	newPerson := model.NewPerson(sid64)
	if errGetPerson := app.store.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	if (newPerson.DiscordID) != "" {
		return errors.Errorf("Discord account already linked to steam account: %d", newPerson.SteamID.Int64())
	}
	newPerson.DiscordID = discordId
	if errSavePerson := app.store.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return consts.ErrInternal
	}
	log.WithFields(log.Fields{"sid64": sid64, "discordId": discordId}).Infof("Discord steamid set")
	return nil
}

// Say is used to send a message to the server via sm_say
func (app *App) Say(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var server model.Server
	if errGetServer := app.store.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		return errors.Errorf("Failed to fetch server: %s", serverName)
	}
	msg := fmt.Sprintf(`sm_say %s`, message)
	rconResponse, errExecRCON := query.ExecRCON(ctx, server, msg)
	if errExecRCON != nil {
		return errExecRCON
	}
	responsePieces := strings.Split(rconResponse, "\n")
	if len(responsePieces) < 2 {
		return errors.Errorf("Invalid response")
	}
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server message sent")
	return nil
}

// CSay is used to send a centered message to the server via sm_csay
func (app *App) CSay(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var (
		servers []model.Server
		err     error
	)
	if serverName == "*" {
		servers, err = app.store.GetServers(ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server model.Server
		if errS := app.store.GetServerByName(ctx, serverName, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", serverName)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, message)
	// TODO check response
	_ = query.RCON(ctx, servers, msg)
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server center message sent")
	return nil
}

// PSay is used to send a private message to a player
func (app *App) PSay(ctx context.Context, database store.Store, author steamid.SID64, target model.StringSID, message string, server *model.Server) error {
	var actualServer *model.Server
	if server != nil {
		actualServer = server
	} else {
		var playerInfo model.PlayerInfo
		// TODO check resp
		_ = app.Find(ctx, target, "", &playerInfo)
		if !playerInfo.Valid || !playerInfo.InGame {
			return consts.ErrUnknownID
		}
		actualServer = playerInfo.Server
	}
	sid, errSid := target.SID64()
	if errSid != nil {
		return errSid
	}
	msg := fmt.Sprintf(`sm_psay "#%s" "%s"`, steamid.SID64ToSID(sid), message)
	_, errExecRCON := query.ExecRCON(ctx, *actualServer, msg)
	if errExecRCON != nil {
		return errors.Errorf("Failed to exec psay command: %v", errExecRCON)
	}
	log.WithFields(log.Fields{"author": author, "server": server.ServerNameShort, "msg": message, "target": sid}).
		Infof("Private message sent")
	return nil
}

// FilterAdd creates a new chat filter using a regex pattern
func (app *App) FilterAdd(ctx context.Context, database store.Store, newPattern *regexp.Regexp, name string) (model.Filter, error) {
	var filter model.Filter
	if errGetFilter := database.GetFilterByName(ctx, name, &filter); errGetFilter != nil {
		if !errors.Is(errGetFilter, store.ErrNoResult) {
			return filter, errors.Wrapf(errGetFilter, "Failed to get parent filter")
		}
		filter.CreatedOn = config.Now()
		filter.FilterName = name
	}
	existing := filter.Patterns
	for _, pat := range existing {
		if pat.String() == newPattern.String() {
			return filter, store.ErrDuplicate
		}
	}
	filter.Patterns = append(filter.Patterns, newPattern)
	if errSave := database.SaveFilter(ctx, &filter); errSave != nil {
		if errSave == store.ErrDuplicate {
			return filter, store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", errSave)
		return filter, consts.ErrInternal
	}
	return filter, nil
}

// FilterDel removed and existing chat filter
func (app *App) FilterDel(ctx context.Context, database store.Store, filterId int64) (bool, error) {
	var filter model.Filter
	if errGetFilter := database.GetFilterByID(ctx, filterId, &filter); errGetFilter != nil {
		return false, errGetFilter
	}
	if errDropFilter := database.DropFilter(ctx, &filter); errDropFilter != nil {
		return false, errDropFilter
	}
	return true, nil
}

// FilterCheck can be used to check if a phrase will match any filters
func (app *App) FilterCheck(message string) []model.Filter {
	if message == "" {
		return nil
	}
	words := strings.Split(strings.ToLower(message), " ")
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	var found []model.Filter
	for _, filter := range wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}
	return found
}

// findFilteredWordMatch checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func findFilteredWordMatch(body string) (string, *model.Filter) {
	if body == "" {
		return "", nil
	}
	words := strings.Split(strings.ToLower(body), " ")
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	for _, filter := range wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				return word, &filter
			}
		}
	}
	return "", nil
}

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date
func (app *App) PersonBySID(ctx context.Context, database store.Store, sid steamid.SID64, person *model.Person) error {
	if errGetPerson := database.GetPersonBySteamID(ctx, sid, person); errGetPerson != nil && errGetPerson != store.ErrNoResult {
		return errGetPerson
	}
	if person.UpdatedOn == person.CreatedOn || time.Since(person.UpdatedOn) > 15*time.Second {
		summary, errSummary := steamweb.PlayerSummaries(steamid.Collection{person.SteamID})
		if errSummary != nil || len(summary) != 1 {
			log.Errorf("Failed to fetch updated profile: %v", errSummary)
			return nil
		}
		var sum = summary[0]
		person.PlayerSummary = &sum
		person.UpdatedOn = config.Now()
		if errSave := database.SavePerson(ctx, person); errSave != nil {
			log.Errorf("Failed to save updated profile: %v", errSummary)
			return nil
		}
		if errGetPersonBySid64 := database.GetPersonBySteamID(ctx, sid, person); errGetPersonBySid64 != nil && errGetPersonBySid64 != store.ErrNoResult {
			return errGetPersonBySid64
		}
	}
	return nil
}

// ResolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout
func ResolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(localCtx, sidStr)
}
