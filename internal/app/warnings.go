package app

import (
	"context"
	"fmt"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func newWarningTracker(log *zap.Logger, store FilterMatchStore, config config.Filter,
	issuedFunc warningsIssuedFunc, exceededFunc warningsExceededFunc,
) *WarningTracker {
	tracker := WarningTracker{
		log:                log.Named("warnTracker"),
		warnings:           make(map[steamid.SID64][]userWarning),
		warningChan:        make(chan newUserWarning),
		wordFilters:        wordFilters{},
		store:              store,
		onWarningsExceeded: exceededFunc,
		onWarning:          issuedFunc,
	}

	tracker.SetConfig(config)

	return &tracker
}

func (w *WarningTracker) SetConfig(config config.Filter) {
	w.config = config
}

type userWarning struct {
	WarnReason    store.Reason
	Message       string
	Matched       string
	MatchedFilter *store.Filter
	CreatedOn     time.Time
}

type WarningTracker struct {
	log                *zap.Logger
	store              FilterMatchStore
	warnings           map[steamid.SID64][]userWarning
	warningChan        chan newUserWarning
	wordFilters        wordFilters
	config             config.Filter
	onWarningsExceeded warningsExceededFunc
	onWarning          warningsIssuedFunc
}

type FilterMatchStore interface {
	SaveFilter(ctx context.Context, filter *store.Filter) error
	GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *store.Person) error
}

func (w *WarningTracker) start(ctx context.Context) {
	ticker := time.NewTicker(w.config.CheckTimeout)

	for {
		select {
		case now := <-ticker.C:
			for steamID := range w.warnings {
				for warnIdx, warning := range w.warnings[steamID] {
					if now.Sub(warning.CreatedOn) > w.config.MatchTimeout {
						if len(w.warnings[steamID]) > 1 {
							w.warnings[steamID] = append(w.warnings[steamID][:warnIdx], w.warnings[steamID][warnIdx+1])
						} else {
							delete(w.warnings, steamID)
						}
					}
				}
			}

			ticker.Reset(w.config.CheckTimeout)

		case newWarn := <-w.warningChan:
			if !newWarn.userMessage.SteamID.Valid() {
				continue
			}

			newWarn.MatchedFilter.TriggerCount++
			if errSave := w.store.SaveFilter(ctx, newWarn.MatchedFilter); errSave != nil {
				w.log.Error("Failed to update filter trigger count", zap.Error(errSave))
			}

			if !newWarn.MatchedFilter.IsEnabled {
				continue
			}

			if !w.config.Dry {
				_, found := w.warnings[newWarn.userMessage.SteamID]
				if !found {
					w.warnings[newWarn.userMessage.SteamID] = []userWarning{}
				}

				w.warnings[newWarn.userMessage.SteamID] = append(w.warnings[newWarn.userMessage.SteamID], newWarn.userWarning)

				var currentWeight int
				for _, existing := range w.warnings[newWarn.userMessage.SteamID] {
					currentWeight += existing.MatchedFilter.Weight
				}

				if currentWeight > w.config.MaxWeight {
					w.log.Info("Warn limit exceeded",
						zap.Int64("sid64", newWarn.userMessage.SteamID.Int64()),
						zap.Int("count", len(w.warnings[newWarn.userMessage.SteamID])))

					if err := w.onWarningsExceeded(ctx, w, newWarn); err != nil {
						w.log.Error("Failed to execute warning exceeded handler", zap.Error(err))
					}
				} else {
					if err := w.onWarning(ctx, w, newWarn); err != nil {
						w.log.Error("Failed to execute warning handler", zap.Error(err))
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

type warningsExceededFunc func(ctx context.Context, w *WarningTracker, newWarning newUserWarning) error

type warningsIssuedFunc func(ctx context.Context, w *WarningTracker, newWarning newUserWarning) error

func onWarningHandler(app WarnApplication) warningsIssuedFunc {
	return func(ctx context.Context, w *WarningTracker, newWarning newUserWarning) error {
		msg := fmt.Sprintf("[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
			"Further offenses will result in mutes/bans")

		if errPSay := app.PSay(ctx, newWarning.userMessage.SteamID, msg); errPSay != nil {
			return errors.Wrap(errPSay, "Failed to send user warning psay message")
		}

		return nil
	}
}

func onWarningExceeded(app WarnApplication) warningsExceededFunc {
	return func(ctx context.Context, tracker *WarningTracker, newWarning newUserWarning) error {
		var (
			errBan   error
			banSteam store.BanSteam
			expIn    = "Permanent"
			expAt    = expIn
		)

		conf := app.config()

		if newWarning.MatchedFilter.Action == store.Ban || newWarning.MatchedFilter.Action == store.Mute {
			duration, errDuration := util.ParseDuration(newWarning.MatchedFilter.Duration)
			if errDuration != nil {
				return errors.Wrap(errDuration, "Failed to parse word filter duration value")
			}

			if errNewBan := store.NewBanSteam(ctx, store.StringSID(conf.General.Owner.String()),
				store.StringSID(newWarning.userMessage.SteamID.String()),
				duration,
				newWarning.WarnReason,
				"",
				"Automatic warning ban",
				store.System,
				0,
				store.NoComm,
				false,
				&banSteam); errNewBan != nil {
				return errors.Wrap(errNewBan, "Failed to create warning ban")
			}
		}

		switch newWarning.MatchedFilter.Action {
		case store.Mute:
			banSteam.BanType = store.NoComm
			errBan = app.BanSteam(ctx, &banSteam)
		case store.Ban:
			banSteam.BanType = store.Banned
			errBan = app.BanSteam(ctx, &banSteam)
		case store.Kick:
			errBan = app.Kick(ctx, store.System, newWarning.userMessage.SteamID, conf.General.Owner, newWarning.WarnReason)
		}

		if errBan != nil {
			return errors.Wrap(errBan, "Failed to apply warning action")
		}

		title := "Language Warning"
		if conf.Filter.Dry {
			title = "[DRYRUN] " + title
		}

		var person store.Person
		if personErr := tracker.store.GetPersonBySteamID(ctx, newWarning.userMessage.SteamID, &person); personErr != nil {
			return errors.Wrap(personErr, "Failed to get person for warning")
		}

		bot := app.bot()
		if bot == nil {
			return nil
		}

		msgEmbed := discord.NewEmbed(conf, title)
		msgEmbed.Embed().
			SetDescription(newWarning.userWarning.Message).
			SetColor(conf.Discord.ColourWarn).
			AddField("Filter ID", fmt.Sprintf("%d", newWarning.MatchedFilter.FilterID)).
			AddField("Matched", newWarning.Matched).
			AddField("Server", newWarning.userMessage.ServerName).InlineAllFields().
			AddField("Pattern", newWarning.MatchedFilter.Pattern)

		msgEmbed.
			AddFieldsSteamID(newWarning.userMessage.SteamID).
			Embed().
			AddField("Name", person.PersonaName)

		if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
			expIn = util.FmtDuration(banSteam.ValidUntil)
			expAt = util.FmtTimeShort(banSteam.ValidUntil)
		}

		msgEmbed.Embed().AddField("Expires In", expIn).
			AddField("Expires At", expAt)

		if conf.Filter.PingDiscord {
			bot.SendPayload(discord.Payload{
				ChannelID: conf.Discord.LogChannelID,
				Embed:     msgEmbed.Message(),
			})
		}

		return nil
	}
}

type WarnApplication interface {
	config() config.Config
	bot() ChatBot
	BanSteam(ctx context.Context, steam *store.BanSteam) error
	Kick(ctx context.Context, origin store.Origin, sid64 steamid.SID64, author steamid.SID64, reason store.Reason) error
	PSay(ctx context.Context, target steamid.SID64, message string) error
}
