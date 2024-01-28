package chat

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/app"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

func NewTracker(log *zap.Logger, matchTimeout time.Duration, dry bool, maxWeight int, onIssue OnIssuedFunc, onExceed OnExceededFunc) *Tracker {
	tracker := Tracker{
		log:                log.Named("warnTracker"),
		warnings:           make(map[steamid.SID64][]domain.UserWarning),
		WarningChan:        make(chan domain.NewUserWarning),
		wordFilters:        app.WordFilters{},
		onWarningsExceeded: onExceed,
		onWarning:          onIssue,
		matchTimeout:       matchTimeout,
		dry:                dry,
		maxWeight:          maxWeight,
		warningMu:          &sync.RWMutex{},
	}

	return &tracker
}

type Tracker struct {
	log                *zap.Logger
	warningMu          *sync.RWMutex
	dry                bool
	maxWeight          int
	warnings           map[steamid.SID64][]domain.UserWarning
	WarningChan        chan domain.NewUserWarning
	wordFilters        app.WordFilters
	onWarningsExceeded OnExceededFunc
	onWarning          OnIssuedFunc
	matchTimeout       time.Duration
}

// State returns a string key so its more easily portable to frontend js w/o using BigInt.
func (w *Tracker) State() map[string][]domain.UserWarning {
	w.warningMu.RLock()
	defer w.warningMu.RUnlock()

	out := make(map[string][]domain.UserWarning)

	for steamID, v := range w.warnings {
		var warnings []domain.UserWarning

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
			if now.Sub(warning.CreatedOn) > w.matchTimeout {
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

func (w *Tracker) trigger(ctx context.Context, newWarn domain.NewUserWarning) {
	if !newWarn.UserMessage.SteamID.Valid() {
		return
	}

	if !w.dry {
		w.warningMu.Lock()

		_, found := w.warnings[newWarn.UserMessage.SteamID]
		if !found {
			w.warnings[newWarn.UserMessage.SteamID] = []domain.UserWarning{}
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

		if currentWeight > w.maxWeight {
			w.log.Info("Warn limit exceeded",
				zap.Int64("sid64", newWarn.UserMessage.SteamID.Int64()),
				zap.Int("count", count), zap.Int("weight", currentWeight))

			if err := w.onWarningsExceeded(ctx, newWarn); err != nil {
				w.log.Error("Failed to execute warning exceeded handler", zap.Error(err))
			}
		} else {
			if err := w.onWarning(ctx, newWarn); err != nil {
				w.log.Error("Failed to execute warning handler", zap.Error(err))
			}
		}
	}
}

func (w *Tracker) Start(ctx context.Context, checkTimeout time.Duration) {
	ticker := time.NewTicker(checkTimeout)

	for {
		select {
		case now := <-ticker.C:
			w.check(now)
			ticker.Reset(checkTimeout)
		case newWarn := <-w.WarningChan:
			w.trigger(ctx, newWarn)
		case <-ctx.Done():
			return
		}
	}
}

type OnExceededFunc func(ctx context.Context, newWarning domain.NewUserWarning) error

type OnIssuedFunc func(ctx context.Context, newWarning domain.NewUserWarning) error
