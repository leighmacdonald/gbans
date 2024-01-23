package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type WarningStore interface {
	SaveFilter(ctx context.Context, filter *model.Filter) error
}

func NewTracker(log *zap.Logger, database WarningStore, conf config.Filter, onIssue OnIssuedFunc, onExceed OnExceededFunc) *Tracker {
	tracker := Tracker{
		log:                log.Named("warnTracker"),
		warnings:           make(map[steamid.SID64][]model.UserWarning),
		WarningChan:        make(chan model.NewUserWarning),
		wordFilters:        WordFilters{},
		db:                 database,
		config:             conf,
		onWarningsExceeded: onExceed,
		onWarning:          onIssue,
		warningMu:          &sync.RWMutex{},
	}

	return &tracker
}

func (w *Tracker) SetConfig(config config.Filter) {
	w.config = config
}

type Tracker struct {
	log                *zap.Logger
	db                 WarningStore
	warningMu          *sync.RWMutex
	warnings           map[steamid.SID64][]model.UserWarning
	WarningChan        chan model.NewUserWarning
	wordFilters        WordFilters
	config             config.Filter
	onWarningsExceeded OnExceededFunc
	onWarning          OnIssuedFunc
}

// State returns a string key so its more easily portable to frontend js w/o using BigInt.
func (w *Tracker) State() map[string][]model.UserWarning {
	w.warningMu.RLock()
	defer w.warningMu.RUnlock()

	out := make(map[string][]model.UserWarning)

	for steamID, v := range w.warnings {
		var warnings []model.UserWarning

		warnings = append(warnings, v...)

		out[steamID.String()] = warnings
	}

	return out
}

func (w *Tracker) check(now time.Time) {
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

func (w *Tracker) trigger(ctx context.Context, newWarn model.NewUserWarning) {
	if !newWarn.UserMessage.SteamID.Valid() {
		return
	}

	newWarn.MatchedFilter.TriggerCount++
	if errSave := w.db.SaveFilter(ctx, newWarn.MatchedFilter); errSave != nil {
		w.log.Error("Failed to update filter trigger count", zap.Error(errSave))
	}

	if !newWarn.MatchedFilter.IsEnabled {
		return
	}

	if !w.config.Dry {
		w.warningMu.Lock()

		_, found := w.warnings[newWarn.UserMessage.SteamID]
		if !found {
			w.warnings[newWarn.UserMessage.SteamID] = []model.UserWarning{}
		}

		var (
			currentWeight = newWarn.MatchedFilter.Weight
			count         int
		)

		for _, existing := range w.warnings[newWarn.UserMessage.SteamID] {
			currentWeight += existing.MatchedFilter.Weight
			count++
		}

		newWarn.CurrentTotal = currentWeight + newWarn.MatchedFilter.Weight

		w.warnings[newWarn.UserMessage.SteamID] = append(w.warnings[newWarn.UserMessage.SteamID], newWarn.UserWarning)

		w.warningMu.Unlock()

		if currentWeight > w.config.MaxWeight {
			w.log.Info("Warn limit exceeded",
				zap.Int64("sid64", newWarn.UserMessage.SteamID.Int64()),
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

func (w *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.config.CheckTimeout)

	for {
		select {
		case now := <-ticker.C:
			w.check(now)
			ticker.Reset(w.config.CheckTimeout)
		case newWarn := <-w.WarningChan:
			w.trigger(ctx, newWarn)
		case <-ctx.Done():
			return
		}
	}
}

type OnExceededFunc func(ctx context.Context, w *Tracker, newWarning model.NewUserWarning) error

type OnIssuedFunc func(ctx context.Context, w *Tracker, newWarning model.NewUserWarning) error

func onWarningHandler(app *App) OnIssuedFunc {
	return func(ctx context.Context, w *Tracker, newWarning model.NewUserWarning) error {
		msg := fmt.Sprintf("[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
			"Further offenses will result in mutes/bans")

		if errPSay := state.PSay(ctx, app.State(), newWarning.UserMessage.SteamID, msg); errPSay != nil {
			return errors.Join(errPSay, state.ErrRCONCommand)
		}

		return nil
	}
}

var (
	ErrFailedToBan     = errors.New("failed to create warning ban")
	ErrWarnActionApply = errors.New("failed to apply warning action")
)

func onWarningExceeded(app *App) OnExceededFunc {
	return func(ctx context.Context, tracker *Tracker, newWarning model.NewUserWarning) error {
		var (
			errBan   error
			banSteam model.BanSteam
		)

		conf := app.Config()

		if newWarning.MatchedFilter.Action == model.Ban || newWarning.MatchedFilter.Action == model.Mute {
			duration, errDuration := util.ParseDuration(newWarning.MatchedFilter.Duration)
			if errDuration != nil {
				return fmt.Errorf("invalid duration: %w", errDuration)
			}

			if errNewBan := model.NewBanSteam(ctx, model.StringSID(conf.General.Owner.String()),
				model.StringSID(newWarning.UserMessage.SteamID.String()),
				duration,
				newWarning.WarnReason,
				"",
				"Automatic warning ban",
				model.System,
				0,
				model.NoComm,
				false,
				&banSteam); errNewBan != nil {
				return errors.Join(errNewBan, ErrFailedToBan)
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
			errBan = state.Kick(ctx, app.state, newWarning.UserMessage.SteamID, newWarning.WarnReason)
		}

		if errBan != nil {
			return errors.Join(errBan, ErrWarnActionApply)
		}

		var person model.Person
		if personErr := app.Store().GetPersonBySteamID(ctx, newWarning.UserMessage.SteamID, &person); personErr != nil {
			return personErr
		}

		if !conf.Filter.PingDiscord {
			return nil
		}

		app.SendPayload(conf.Discord.LogChannelID, discord.WarningMessage(newWarning, banSteam, person))

		return nil
	}
}
