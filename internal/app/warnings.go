package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func newWarningTracker(log *zap.Logger, store store.Store, config config.Filter,
	issuedFunc warningsIssuedFunc, exceededFunc warningsExceededFunc,
) *WarningTracker {
	tracker := WarningTracker{
		log:                log.Named("warnTracker"),
		warnings:           make(map[steamid.SID64][]userWarning),
		warningChan:        make(chan newUserWarning),
		wordFilters:        wordFilters{},
		db:                 store,
		onWarningsExceeded: exceededFunc,
		onWarning:          issuedFunc,
		warningMu:          &sync.RWMutex{},
	}

	tracker.SetConfig(config)

	return &tracker
}

func (w *WarningTracker) SetConfig(config config.Filter) {
	w.config = config
}

type newUserWarning struct {
	userMessage model.PersonMessage
	userWarning
}

type userWarning struct {
	WarnReason    model.Reason  `json:"warn_reason"`
	Message       string        `json:"message"`
	Matched       string        `json:"matched"`
	MatchedFilter *model.Filter `json:"matched_filter"`
	CreatedOn     time.Time     `json:"created_on"`
	Personaname   string        `json:"personaname"`
	Avatar        string        `json:"avatar"`
	ServerName    string        `json:"server_name"`
	ServerID      int           `json:"server_id"`
	SteamID       string        `json:"steam_id"`
	CurrentTotal  int           `json:"current_total"`
}

type WarningTracker struct {
	log                *zap.Logger
	db                 store.Store
	warningMu          *sync.RWMutex
	warnings           map[steamid.SID64][]userWarning
	warningChan        chan newUserWarning
	wordFilters        wordFilters
	config             config.Filter
	onWarningsExceeded warningsExceededFunc
	onWarning          warningsIssuedFunc
}

// state returns a string key so its more easily portable to frontend js w/o using BigInt.
func (w *WarningTracker) state() map[string][]userWarning {
	w.warningMu.RLock()
	defer w.warningMu.RUnlock()

	out := make(map[string][]userWarning)

	for steamID, v := range w.warnings {
		var warnings []userWarning

		warnings = append(warnings, v...)

		out[steamID.String()] = warnings
	}

	return out
}

func (w *WarningTracker) check(now time.Time) {
	w.warningMu.Lock()
	defer w.warningMu.Unlock()

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

		var newSum int
		for idx := range w.warnings[steamID] {
			newSum += w.warnings[steamID][idx].MatchedFilter.Weight
			w.warnings[steamID][idx].CurrentTotal = newSum
		}
	}
}

func (w *WarningTracker) trigger(ctx context.Context, newWarn newUserWarning) {
	if !newWarn.userMessage.SteamID.Valid() {
		return
	}

	newWarn.MatchedFilter.TriggerCount++
	if errSave := store.SaveFilter(ctx, w.db, newWarn.MatchedFilter); errSave != nil {
		w.log.Error("Failed to update filter trigger count", zap.Error(errSave))
	}

	if !newWarn.MatchedFilter.IsEnabled {
		return
	}

	if !w.config.Dry {
		w.warningMu.Lock()

		_, found := w.warnings[newWarn.userMessage.SteamID]
		if !found {
			w.warnings[newWarn.userMessage.SteamID] = []userWarning{}
		}

		var (
			currentWeight = newWarn.MatchedFilter.Weight
			count         int
		)

		for _, existing := range w.warnings[newWarn.userMessage.SteamID] {
			currentWeight += existing.MatchedFilter.Weight
			count++
		}

		newWarn.CurrentTotal = currentWeight + newWarn.MatchedFilter.Weight

		w.warnings[newWarn.userMessage.SteamID] = append(w.warnings[newWarn.userMessage.SteamID], newWarn.userWarning)

		w.warningMu.Unlock()

		if currentWeight > w.config.MaxWeight {
			w.log.Info("Warn limit exceeded",
				zap.Int64("sid64", newWarn.userMessage.SteamID.Int64()),
				zap.Int("count", count), zap.Int("weight", currentWeight))

			if err := w.onWarningsExceeded(ctx, w, newWarn); err != nil {
				w.log.Error("Failed to execute warning exceeded handler", zap.Error(err))
			}
		} else {
			if err := w.onWarning(ctx, w, newWarn); err != nil {
				w.log.Error("Failed to execute warning handler", zap.Error(err))
			}
		}
	}
}

func (w *WarningTracker) start(ctx context.Context) {
	ticker := time.NewTicker(w.config.CheckTimeout)

	for {
		select {
		case now := <-ticker.C:
			w.check(now)
			ticker.Reset(w.config.CheckTimeout)
		case newWarn := <-w.warningChan:
			w.trigger(ctx, newWarn)
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
			banSteam model.BanSteam
			expIn    = "Permanent"
			expAt    = expIn
		)

		conf := app.config()

		if newWarning.MatchedFilter.Action == model.Ban || newWarning.MatchedFilter.Action == model.Mute {
			duration, errDuration := util.ParseDuration(newWarning.MatchedFilter.Duration)
			if errDuration != nil {
				return errors.Wrap(errDuration, "Failed to parse word filter duration value")
			}

			if errNewBan := model.NewBanSteam(ctx, model.StringSID(conf.General.Owner.String()),
				model.StringSID(newWarning.userMessage.SteamID.String()),
				duration,
				newWarning.WarnReason,
				"",
				"Automatic warning ban",
				model.System,
				0,
				model.NoComm,
				false,
				&banSteam); errNewBan != nil {
				return errors.Wrap(errNewBan, "Failed to create warning ban")
			}
		}

		switch newWarning.MatchedFilter.Action {
		case model.Mute:
			banSteam.BanType = model.NoComm
			errBan = app.BanSteam(ctx, &banSteam)
		case model.Ban:
			banSteam.BanType = model.Banned
			errBan = app.BanSteam(ctx, &banSteam)
		case model.Kick:
			errBan = app.Kick(ctx, model.System, newWarning.userMessage.SteamID, conf.General.Owner, newWarning.WarnReason)
		}

		if errBan != nil {
			return errors.Wrap(errBan, "Failed to apply warning action")
		}

		title := "Language Warning"
		if conf.Filter.Dry {
			title = "[DRYRUN] " + title
		}

		var person model.Person
		if personErr := store.GetPersonBySteamID(ctx, tracker.db, newWarning.userMessage.SteamID, &person); personErr != nil {
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
	BanSteam(ctx context.Context, steam *model.BanSteam) error
	Kick(ctx context.Context, origin model.Origin, sid64 steamid.SID64, author steamid.SID64, reason model.Reason) error
	PSay(ctx context.Context, target steamid.SID64, message string) error
}
