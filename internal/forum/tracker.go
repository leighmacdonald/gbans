package forum

import (
	"context"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type Tracker struct {
	activityMu sync.RWMutex
	activity   []domain.ForumActivity
}

func NewTracker() *Tracker {
	return &Tracker{
		activity: make([]domain.ForumActivity, 0),
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
