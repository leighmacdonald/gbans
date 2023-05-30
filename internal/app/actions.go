package app

import (
	"context"
	gerrors "errors"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
)

var ErrNoUserFound = errors.New("No user found")
var ErrInvalidAuthorSID = errors.New("Invalid author steam id")
var ErrInvalidTargetSID = errors.New("Invalid author steam id")

// OnFindExec is a helper function used to execute rcon commands against any players found in the query
func OnFindExec(ctx context.Context, findOpts state.FindOpts, onFoundCmd func(info state.PlayerServerInfo) string) error {
	players, found := state.Find(findOpts)
	if !found {
		return ErrNoUserFound
	}
	var err error
	for _, player := range players {
		var server store.Server
		if errServer := store.GetServer(ctx, player.ServerId, &server); errServer != nil {
			err = gerrors.Join(err, errServer)
			continue
		}
		cmd := onFoundCmd(player)
		_, errExecRCON := query.ExecRCON(ctx, server.Addr(), server.RCON, cmd)
		if errExecRCON != nil {
			err = gerrors.Join(err, errExecRCON)
			continue
		}
	}
	return err
}

// Kick will kick the steam id from whatever server it is connected to.
func Kick(ctx context.Context, origin store.Origin, target steamid.SID64, author steamid.SID64, reason store.Reason) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}
	return OnFindExec(ctx, state.FindOpts{SteamID: target}, func(info state.PlayerServerInfo) string {
		return fmt.Sprintf("sm_kick #%d %s", info.Player.UserID, reason)
	})
}

// Silence will gag & mute a player
func Silence(ctx context.Context, origin store.Origin, target steamid.SID64, author steamid.SID64,
	reason store.Reason) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}
	return OnFindExec(ctx, state.FindOpts{SteamID: target}, func(info state.PlayerServerInfo) string {
		return fmt.Sprintf(`sm_silence "#%s" %s`, steamid.SID64ToSID(info.Player.SID), reason.String())
	})
}

// Say is used to send a message to the server via sm_say
func Say(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var server store.Server
	if errGetServer := store.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		return errors.Errorf("Failed to fetch server: %s", serverName)
	}
	msg := fmt.Sprintf(`sm_say %s`, message)
	rconResponse, errExecRCON := query.ExecRCON(ctx, server.Addr(), server.RCON, msg)
	if errExecRCON != nil {
		return errExecRCON
	}
	responsePieces := strings.Split(rconResponse, "\n")
	if len(responsePieces) < 2 {
		return errors.Errorf("Invalid response")
	}
	logger.Info("Server message sent", zap.Int64("author", author.Int64()), zap.String("msg", message))
	return nil
}

// CSay is used to send a centered message to the server via sm_csay
func CSay(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var (
		servers []store.Server
		err     error
	)
	if serverName == "*" {
		servers, err = store.GetServers(ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server store.Server
		if errS := store.GetServerByName(ctx, serverName, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", serverName)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, message)
	// TODO check response
	_ = query.RCON(ctx, logger, servers, msg)
	logger.Info("Server center message sent", zap.Int64("author", author.Int64()),
		zap.String("msg", message), zap.Int("servers", len(servers)))
	return nil
}

// PSay is used to send a private message to a player
func PSay(ctx context.Context, author steamid.SID64, target steamid.SID64, message string) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}
	return OnFindExec(ctx, state.FindOpts{SteamID: target}, func(info state.PlayerServerInfo) string {
		return fmt.Sprintf(`sm_psay "#%s" "%s"`, steamid.SID64ToSID(target), message)
	})
}

// SetSteam is used to associate a discordutil user with either steam id. This is used
// instead of requiring users to link their steam account to discordutil itself. It also
// means the bot does not require more privileged intents.
func SetSteam(ctx context.Context, sid64 steamid.SID64, discordId string) error {
	newPerson := store.NewPerson(sid64)
	if errGetPerson := store.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	if (newPerson.DiscordID) != "" {
		return errors.Errorf("Discord account already linked to steam account: %d", newPerson.SteamID.Int64())
	}
	newPerson.DiscordID = discordId
	if errSavePerson := store.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return consts.ErrInternal
	}
	logger.Info("Discord steamid set", zap.Int64("sid64", sid64.Int64()), zap.String("discordId", discordId))
	return nil
}

// FilterAdd creates a new chat filter using a regex pattern
func FilterAdd(ctx context.Context, filter *store.Filter) error {
	if errSave := store.SaveFilter(ctx, filter); errSave != nil {
		if errSave == store.ErrDuplicate {
			return store.ErrDuplicate
		}
		logger.Error("Error saving filter word", zap.Error(errSave))
		return consts.ErrInternal
	}
	filter.Init()
	wordFiltersMu.Lock()
	wordFilters = append(wordFilters, *filter)
	wordFiltersMu.Unlock()

	return nil
}

// FilterDel removed and existing chat filter
func FilterDel(ctx context.Context, filterId int64) (bool, error) {
	var filter store.Filter
	if errGetFilter := store.GetFilterByID(ctx, filterId, &filter); errGetFilter != nil {
		return false, errGetFilter
	}
	if errDropFilter := store.DropFilter(ctx, &filter); errDropFilter != nil {
		return false, errDropFilter
	}
	wordFiltersMu.Lock()
	var valid []store.Filter
	for _, f := range wordFilters {
		if f.FilterID == filterId {
			continue
		}
		valid = append(valid, f)
	}
	wordFilters = valid
	wordFiltersMu.Unlock()
	return true, nil
}

// FilterCheck can be used to check if a phrase will match any filters
func FilterCheck(message string) []store.Filter {
	if message == "" {
		return nil
	}
	words := strings.Split(strings.ToLower(message), " ")
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	var found []store.Filter
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
func findFilteredWordMatch(body string) (string, *store.Filter) {
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

//// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date
//func (app *App) PersonBySID(ctx context.Context, sid steamid.SID64, person *model.Person) error {
//	if errGetPerson := app.store.GetPersonBySteamID(ctx, sid, person); errGetPerson != nil && errGetPerson != store.ErrNoResult {
//		return errGetPerson
//	}
//	if person.UpdatedOn == person.CreatedOn || time.Since(person.UpdatedOn) > 15*time.Second {
//		summary, errSummary := steamweb.PlayerSummaries(steamid.Collection{person.SteamID})
//		if errSummary != nil || len(summary) != 1 {
//			app.logger.Error("Failed to fetch updated profile", zap.Error(errSummary))
//			return nil
//		}
//		var sum = summary[0]
//		person.PlayerSummary = &sum
//		person.UpdatedOn = config.Now()
//		if errSave := app.store.SavePerson(ctx, person); errSave != nil {
//			app.logger.Error("Failed to save updated profile", zap.Error(errSummary))
//			return nil
//		}
//		if errGetPersonBySid64 := app.store.GetPersonBySteamID(ctx, sid, person); errGetPersonBySid64 != nil &&
//			errGetPersonBySid64 != store.ErrNoResult {
//			return errGetPersonBySid64
//		}
//	}
//	return nil
//}
