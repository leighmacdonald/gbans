package app

import (
	"context"
	gerrors "errors"
	"fmt"
	"strings"

	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrNoUserFound      = errors.New("No user found")
	ErrInvalidAuthorSID = errors.New("Invalid author steam id")
	ErrInvalidTargetSID = errors.New("Invalid author steam id")
)

// OnFindExec is a helper function used to execute rcon commands against any players found in the query.
func (app *App) OnFindExec(ctx context.Context, findOpts FindOpts, onFoundCmd func(info PlayerServerInfo) string) error {
	players, found := app.Find(findOpts)
	if !found {
		return ErrNoUserFound
	}
	var err error
	for _, player := range players {
		var server store.Server
		if errServer := app.db.GetServer(ctx, player.ServerID, &server); errServer != nil {
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
func (app *App) Kick(ctx context.Context, _ store.Origin, target steamid.SID64, author steamid.SID64, reason store.Reason) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}

	return app.OnFindExec(ctx, FindOpts{SteamID: target}, func(info PlayerServerInfo) string {
		return fmt.Sprintf("sm_kick #%d %s", info.Player.UserID, store.ReasonString(reason))
	})
}

// Silence will gag & mute a player.
func (app *App) Silence(ctx context.Context, _ store.Origin, target steamid.SID64, author steamid.SID64,
	reason store.Reason,
) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}

	return app.OnFindExec(ctx, FindOpts{SteamID: target}, func(info PlayerServerInfo) string {
		return fmt.Sprintf(`sm_silence "#%s" %s`, steamid.SID64ToSID(info.Player.SID), store.ReasonString(reason))
	})
}

// Say is used to send a message to the server via sm_say.
func (app *App) Say(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var server store.Server
	if errGetServer := app.db.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		return errors.Errorf("Failed to fetch server: %s", serverName)
	}
	msg := fmt.Sprintf(`sm_say %s`, message)
	rconResponse, errExecRCON := query.ExecRCON(ctx, server.Addr(), server.RCON, msg)
	if errExecRCON != nil {
		return errors.Wrapf(errExecRCON, "Failed to exec say command")
	}
	responsePieces := strings.Split(rconResponse, "\n")
	if len(responsePieces) < 2 {
		return errors.Errorf("Invalid response")
	}
	app.log.Info("Server message sent", zap.Int64("author", author.Int64()), zap.String("msg", message))

	return nil
}

// CSay is used to send a centered message to the server via sm_csay.
func (app *App) CSay(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	var (
		servers []store.Server
		err     error
	)
	if serverName == "*" {
		servers, err = app.db.GetServers(ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server store.Server
		if errS := app.db.GetServerByName(ctx, serverName, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", serverName)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, message)
	// TODO check response
	_ = query.RCON(ctx, app.log, servers, msg)
	app.log.Info("Server center message sent", zap.Int64("author", author.Int64()),
		zap.String("msg", message), zap.Int("servers", len(servers)))

	return nil
}

// PSay is used to send a private message to a player.
func (app *App) PSay(ctx context.Context, author steamid.SID64, target steamid.SID64, message string) error {
	if !author.Valid() {
		return ErrInvalidAuthorSID
	}
	if !target.Valid() {
		return ErrInvalidTargetSID
	}

	return app.OnFindExec(ctx, FindOpts{SteamID: target}, func(info PlayerServerInfo) string {
		return fmt.Sprintf(`sm_psay "#%s" "%s"`, steamid.SID64ToSID(target), message)
	})
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the bot does not require more privileged intents.
func (app *App) SetSteam(ctx context.Context, sid64 steamid.SID64, discordID string) error {
	newPerson := store.NewPerson(sid64)
	if errGetPerson := app.db.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	if (newPerson.DiscordID) != "" {
		return errors.Errorf("Discord account already linked to steam account: %d", newPerson.SteamID.Int64())
	}
	newPerson.DiscordID = discordID
	if errSavePerson := app.db.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return consts.ErrInternal
	}
	app.log.Info("Discord steamid set", zap.Int64("sid64", sid64.Int64()), zap.String("discordId", discordID))

	return nil
}

// FilterAdd creates a new chat filter using a regex pattern.
func (app *App) FilterAdd(ctx context.Context, filter *store.Filter) error {
	if errSave := app.db.SaveFilter(ctx, filter); errSave != nil {
		if errors.Is(errSave, store.ErrDuplicate) {
			return store.ErrDuplicate
		}
		app.log.Error("Error saving filter word", zap.Error(errSave))

		return consts.ErrInternal
	}
	filter.Init()
	app.wordFilters.Lock()
	app.wordFilters.wordFilters = append(app.wordFilters.wordFilters, *filter)
	app.wordFilters.Unlock()

	return nil
}

// FilterDel removed and existing chat filter.
func (app *App) FilterDel(ctx context.Context, filterID int64) (bool, error) {
	var filter store.Filter
	if errGetFilter := app.db.GetFilterByID(ctx, filterID, &filter); errGetFilter != nil {
		return false, errors.Wrap(errGetFilter, "Failed to get filter")
	}

	if errDropFilter := app.db.DropFilter(ctx, &filter); errDropFilter != nil {
		return false, errors.Wrapf(errDropFilter, "Failed to drop filter")
	}

	app.wordFilters.Lock()
	defer app.wordFilters.Unlock()

	var valid []store.Filter //nolint:prealloc
	for _, f := range app.wordFilters.wordFilters {
		if f.FilterID == filterID {
			continue
		}
		valid = append(valid, f)
	}

	app.wordFilters.wordFilters = valid

	return true, nil
}

// FilterCheck can be used to check if a phrase will match any filters.
func (app *App) FilterCheck(message string) []store.Filter {
	if message == "" {
		return nil
	}
	words := strings.Split(strings.ToLower(message), " ")
	app.wordFilters.RLock()
	defer app.wordFilters.RUnlock()
	var found []store.Filter
	for _, filter := range app.wordFilters.wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}

	return found
}
