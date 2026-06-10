package stats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
)

const (
	MinPlayers = 4
	MinDuraion = 300
)

var (
	ErrInvalidState  = errors.New("invalid demo state")
	ErrInvalidBucket = errors.New("invalid stat bucket")
	ErrJob           = errors.New("stat update job error")
)

type Stats struct {
	repo Repository
	maps maps.Maps
}

func New(repo Repository, maps maps.Maps) Stats {
	return Stats{repo: repo, maps: maps}
}

func (s Stats) StartRefreshHandler(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return errors.Join(err, ErrJob)
	}
	// 00:00
	// scheduler.NewJob(gocron.CronJob("0 0 * * *", false), gocron.NewTask(s.updateAlltimeViews(ctx)))
	// 00:05
	if _, err := scheduler.NewJob(gocron.CronJob("5 0 * * *", false), gocron.NewTask(s.updateDailyViews(ctx))); err != nil {
		return errors.Join(err, ErrJob)
	}
	// Sunday 00:00
	// scheduler.NewJob(gocron.CronJob("0 0 * * 0", false), gocron.NewTask(s.updateDailyViews(ctx)))
	// 1st of month 00:00
	// scheduler.NewJob(gocron.CronJob("0 0 1 * *", false), gocron.NewTask(s.updateDailyViews(ctx)))

	scheduler.Start()

	<-ctx.Done()

	if errShutdown := scheduler.Shutdown(); err != nil {
		return errors.Join(errShutdown, ErrJob)
	}

	return nil
}

func (s Stats) updateDailyViews(ctx context.Context) func() {
	return func() {
		slog.Debug("Refreshing daily stat views")
		for _, viewName := range []string{"stats_summary_daily", "stats_summary_daily_weapons", "stats_summary_daily_classes"} {
			if err := s.repo.RefreshMaterializedView(ctx, viewName); err != nil {
				slog.Error("Failed to refresh view", slog.String("view", viewName), slog.String("error", err.Error()))

				return
			}
		}
	}
}

func (s Stats) Bucket(ctx context.Context, bucketID int32) (*Bucket, error) {
	if bucketID <= 0 {
		return nil, ErrInvalidBucket
	}

	return s.repo.GetBucket(ctx, bucketID)
}

func (s Stats) Match(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	return s.repo.Match(ctx, matchID)
}

func (s Stats) Import(ctx context.Context, serverID int32, demoID int32, demo *demoparse.Demo, timeStart time.Time) (*uuid.UUID, error) {
	if demo.DemoType != demoparse.HL2Demo {
		return nil, fmt.Errorf("%w: invalid demo type", ErrInvalidState)
	}

	if demo.Server == "" {
		return nil, fmt.Errorf("%w: invalid server name", ErrInvalidState)
	}

	if demo.Filename == "" {
		return nil, fmt.Errorf("%w: invalid file name", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if len(demo.Rounds) == 0 {
		return nil, fmt.Errorf("%w not enough rounds", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if demo.Map == "" {
		return nil, fmt.Errorf("%w: empty map invalid", ErrInvalidState)
	}

	mapInfo, errMap := s.maps.Get(ctx, demo.Map)
	if errMap != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidState, errMap)
	}

	matchID, errMatch := s.repo.CreateMatch(ctx, serverID, demoID, demo, timeStart, mapInfo, nil)
	if errMatch != nil {
		return nil, errMatch
	}

	match, errMatch := s.repo.Match(ctx, matchID)
	if errMatch != nil {
		return nil, errMatch
	}
	slog.Info("Got match", slog.String("match", match.Hostname))

	return &matchID, nil
}
