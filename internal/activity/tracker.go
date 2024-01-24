package activity

import (
	"context"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"go.uber.org/zap"
)

type Tracker struct {
	activityMu *sync.RWMutex
	activity   []domain.ForumActivity
	log        *zap.Logger
}

func NewTracker(log *zap.Logger) *Tracker {
	return &Tracker{
		activityMu: &sync.RWMutex{},
		activity:   make([]domain.ForumActivity, 0),
		log:        log.Named("Tracker"),
	}
}

func (tracker *Tracker) Touch(person domain.UserProfile) {
	if !person.SteamID.Valid() {
		return
	}

	valid := []domain.ForumActivity{{LastActivity: time.Now(), Person: person}}

	tracker.activityMu.Lock()
	defer tracker.activityMu.Unlock()

	for _, activity := range tracker.activity {
		if activity.Person.SteamID == person.SteamID {
			continue
		}

		valid = append(valid, activity)
	}

	tracker.activity = valid
}

func (tracker *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)

	for {
		select {
		case <-ticker.C:
			var current []domain.ForumActivity

			tracker.activityMu.Lock()

			for _, entry := range tracker.activity {
				if entry.Expired() {
					tracker.log.Debug("Player forum activity expired", zap.Int64("steam_id", entry.Person.SteamID.Int64()))

					continue
				}

				current = append(current, entry)
			}

			tracker.activity = current

			tracker.activityMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (tracker *Tracker) Current() []domain.ForumActivity {
	tracker.activityMu.RLock()
	defer tracker.activityMu.RUnlock()

	var activity []domain.ForumActivity

	activity = append(activity, tracker.activity...)

	return activity
}
