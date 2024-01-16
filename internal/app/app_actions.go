package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// OnFindExec is a helper function used to execute rcon commands against any players found in the query.
func (app *App) OnFindExec(_ context.Context, findOpts findOpts, onFoundCmd func(info playerServerInfo) string) error {
	state := app.state.current()

	players := state.find(findOpts)
	if len(players) == 0 {
		return consts.ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		for _, server := range state {
			if player.ServerID == server.ServerID {
				resp, errRcon := app.state.rcon(player.ServerID, onFoundCmd(player))
				if errRcon != nil {
					app.log.Error("Bad rcon response", zap.Error(errRcon))

					continue
				}

				app.log.Debug("Successful rcon response", zap.String("resp", resp))
			}
		}
	}

	return err
}

// Kick will kick the steam id from whatever server it is connected to.
func (app *App) Kick(ctx context.Context, _ store.Origin, target steamid.SID64, author steamid.SID64, reason store.Reason) error {
	if !author.Valid() {
		return consts.ErrInvalidAuthorSID
	}

	if !target.Valid() {
		return consts.ErrInvalidTargetSID
	}

	var (
		server   []string
		serverMu = &sync.RWMutex{}
	)

	if errExec := app.OnFindExec(ctx, findOpts{SteamID: target}, func(info playerServerInfo) string {
		serverMu.Lock()
		server = append(server, fmt.Sprintf("%d", info.ServerID))
		serverMu.Unlock()

		return fmt.Sprintf("sm_kick #%d %s", info.Player.UserID, reason.String())
	}); errExec != nil {
		return errExec
	}

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "User Kicked Successfully")
	msgEmbed.
		Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("servers", strings.Join(fp.Uniq(server), ","))

	var targetUser store.Person
	if errTargetUser := app.db.GetOrCreatePersonBySteamID(ctx, target, &targetUser); errTargetUser != nil {
		return errors.Wrap(errTargetUser, "Failed to get target")
	}

	var authorUser store.Person
	if errTargetUser := app.db.GetOrCreatePersonBySteamID(ctx, author, &authorUser); errTargetUser != nil {
		return errors.Wrap(errTargetUser, "Failed to get author")
	}

	msgEmbed.AddTargetPerson(targetUser)
	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Embed().MessageEmbed})

	return nil
}

// Silence will gag & mute a player.
func (app *App) Silence(ctx context.Context, _ store.Origin, target steamid.SID64, author steamid.SID64,
	reason store.Reason,
) error {
	if !author.Valid() {
		return consts.ErrInvalidAuthorSID
	}

	if !target.Valid() {
		return consts.ErrInvalidTargetSID
	}

	var (
		users   []string
		usersMu = &sync.RWMutex{}
	)

	if errExec := app.OnFindExec(ctx, findOpts{SteamID: target}, func(info playerServerInfo) string {
		usersMu.Lock()
		users = append(users, info.Player.Name)
		usersMu.Unlock()

		return fmt.Sprintf(`sm_silence "#%s" %s`, steamid.SID64ToSID(info.Player.SID), reason.String())
	}); errExec != nil {
		return errExec
	}

	conf := app.config()
	msgEmbed := discord.NewEmbed(conf, "User Silenced Successfully")
	msgEmbed.
		Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("users", strings.Join(fp.Uniq(users), ","))

	var targetUser store.Person
	if errTargetUser := app.db.GetOrCreatePersonBySteamID(ctx, target, &targetUser); errTargetUser != nil {
		return errors.Wrap(errTargetUser, "Failed to get target")
	}

	var authorUser store.Person
	if errTargetUser := app.db.GetOrCreatePersonBySteamID(ctx, author, &authorUser); errTargetUser != nil {
		return errors.Wrap(errTargetUser, "Failed to get author")
	}

	msgEmbed.AddTargetPerson(targetUser)
	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message()})

	return nil
}

// Say is used to send a message to the server via sm_say.
func (app *App) Say(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	state := app.state.current()
	servers := state.serverIDsByName(serverName, true)

	if len(servers) == 0 {
		return errUnknownServer
	}

	app.state.broadcast(servers, fmt.Sprintf(`sm_say %s`, message))
	app.log.Info("Server Message Sent Successfully", zap.Int64("author", author.Int64()), zap.String("msg", message))

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "Message sent successfully")
	msgEmbed.
		Embed().
		SetDescription(message).
		SetColor(conf.Discord.ColourSuccess).
		AddField("servers", fmt.Sprintf("%d", len(servers)))

	var authorUser store.Person
	if errTargetUser := app.db.GetOrCreatePersonBySteamID(ctx, author, &authorUser); errTargetUser != nil {
		return errors.Wrap(errTargetUser, "Failed to get author")
	}

	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message()})

	return nil
}

// CSay is used to send a centered message to the server via sm_csay.
func (app *App) CSay(ctx context.Context, author steamid.SID64, serverName string, message string) error {
	state := app.state.current()
	servers := state.serverIDsByName(serverName, true)

	if len(servers) == 0 {
		return errUnknownServer
	}

	app.state.broadcast(servers, fmt.Sprintf(`sm_csay %s`, message))

	app.log.Info("Server Center Message Sent Successfully", zap.Int64("author", author.Int64()),
		zap.String("msg", message), zap.Int("servers", len(servers)))

	conf := app.config()
	msgEmbed := discord.NewEmbed(conf, "center Message Sent Successfully")
	msgEmbed.
		Embed().
		SetDescription(message).
		SetColor(conf.Discord.ColourSuccess).
		AddField("servers", fmt.Sprintf("%d", len(servers)))

	var authorUser store.Person
	if errAuthor := app.db.GetOrCreatePersonBySteamID(ctx, author, &authorUser); errAuthor != nil {
		return errors.Wrap(errAuthor, "Failed to get author")
	}

	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message()})

	return nil
}

// PSay is used to send a private message to a player.
func (app *App) PSay(ctx context.Context, target steamid.SID64, message string) error {
	if !target.Valid() {
		return consts.ErrInvalidTargetSID
	}

	if errExec := app.OnFindExec(ctx, findOpts{SteamID: target}, func(info playerServerInfo) string {
		return fmt.Sprintf(`sm_psay "#%s" "%s"`, steamid.SID64ToSID(target), message)
	}); errExec != nil {
		return errExec
	}

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "Private Message Sent Successfully")
	msgEmbed.
		Embed().
		SetDescription(message).
		SetColor(conf.Discord.ColourSuccess)

	app.discord.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message()})

	return nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
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

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "User Set Discord ID Successfully")
	msgEmbed.Embed().SetColor(conf.Discord.ColourSuccess)

	var authorUser store.Person
	if errAuthor := app.db.GetOrCreatePersonBySteamID(ctx, sid64, &authorUser); errAuthor != nil {
		return errors.Wrap(errAuthor, "Failed to get author")
	}

	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID,
		Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
	})

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

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "New Word Filter Created")
	msgEmbed.
		Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("Pattern", filter.Pattern).
		AddField("filter_id", fmt.Sprintf("%d", filter.FilterID))

	var authorUser store.Person
	if errAuthor := app.db.GetOrCreatePersonBySteamID(ctx, filter.AuthorID, &authorUser); errAuthor != nil {
		return errors.Wrap(errAuthor, "Failed to get author")
	}

	msgEmbed.AddAuthorPersonInfo(authorUser)

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message(),
	})

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

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "Filter Deleted")
	msgEmbed.
		Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("Pattern", filter.Pattern).
		AddField("filter_id", fmt.Sprintf("%d", filter.FilterID))

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message(),
	})

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
