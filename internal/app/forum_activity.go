package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"go.uber.org/zap"
	"sync"
	"time"
)

type forumActivity struct {
	person       model.UserProfile
	lastActivity time.Time
}

func (activity forumActivity) expired() bool {
	return time.Since(activity.lastActivity) > time.Minute*5
}

type activityTracker struct {
	activityMu *sync.RWMutex
	activity   []forumActivity
	log        *zap.Logger
}

func newForumActivity(log *zap.Logger) *activityTracker {
	return &activityTracker{
		activityMu: &sync.RWMutex{},
		activity:   make([]forumActivity, 0),
		log:        log.Named("activityTracker"),
	}
}

func (tracker *activityTracker) touch(person model.UserProfile) {
	if !person.SteamID.Valid() {
		return
	}

	valid := []forumActivity{{lastActivity: time.Now(), person: person}}

	tracker.activityMu.Lock()
	defer tracker.activityMu.Unlock()

	for _, activity := range tracker.activity {
		if activity.person.SteamID == person.SteamID {
			continue
		}

		valid = append(valid, activity)
	}

	tracker.activity = valid
}

func (tracker *activityTracker) start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)

	for {
		select {
		case <-ticker.C:
			var current []forumActivity

			tracker.activityMu.Lock()

			for _, entry := range tracker.activity {
				if entry.expired() {
					tracker.log.Debug("Player forum activity expired", zap.Int64("steam_id", entry.person.SteamID.Int64()))

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

func (tracker *activityTracker) current() []forumActivity {
	tracker.activityMu.RLock()
	defer tracker.activityMu.RUnlock()

	var activity []forumActivity

	activity = append(activity, tracker.activity...)

	return activity
}
