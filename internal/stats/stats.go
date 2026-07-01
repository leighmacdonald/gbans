package stats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	MinPlayers = 1
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

func (s Stats) Delete(ctx context.Context, demoID int32) error {
	return s.repo.Delete(ctx, demoID)
}

type VariantStats struct {
	Rank    uint64
	SteamID steamid.SteamID
	Variant string

	Kills               uint64
	Assists             uint64
	Deaths              uint64
	PostroundKills      uint64
	PostroundAssists    uint64
	PostroundDeaths     uint64
	Damage              uint64
	DamageTaken         uint64
	Dominations         uint64
	Dominated           uint64
	Revenges            uint64
	Revenged            uint64
	Airshots            uint64
	HeadshotKills       uint64
	BackstabKills       uint64
	Headshots           uint64
	Backstabs           uint64
	WasHeadshot         uint64
	WasBackstabbed      uint64
	PreroundHealing     uint64
	Healing             uint64
	PostroundHealing    uint64
	Drops               uint64
	NearFullChargeDeath uint64
	ChargesUber         uint64
	ChargesKritz        uint64
	ChargesVacc         uint64
	ChargesQuickfix     uint64
	Shots               uint64
	Hits                uint64
	ObjectsBuilt        uint64
	ObjectsDestroyed    uint64
	Captures            uint64
	CapturesBlocked     uint64
}

type OverallStats struct {
	Rank    uint64
	SteamID steamid.SteamID
	Variant string

	MVP                 bool
	TickStart           uint64
	TickEnd             uint64
	Points              uint64
	ConnectionCount     uint64
	BonusPoints         uint64
	Kills               uint64
	Assists             uint64
	Deaths              uint64
	PostroundKills      uint64
	PostroundAssists    uint64
	PostroundDeaths     uint64
	PostroundHealing    uint64
	Healing             uint64
	PreroundHealing     uint64
	Drops               uint64
	NearFullChargeDeath uint64
	ChargesUber         uint64
	ChargesKritz        uint64
	ChargesVacc         uint64
	ChargesQuickfix     uint64
	Damage              uint64
	DamageTaken         uint64
	Dominations         uint64
	Dominated           uint64
	Revenges            uint64
	Revenged            uint64
	Airshots            uint64
	Headshots           uint64
	HeadshotKills       uint64
	Backstabs           uint64
	BackstabKills       uint64
	WasHeadshot         uint64
	WasBackstabbed      uint64
	Shots               uint64
	Hits                uint64
	ScoreboardKills     uint64
	ScoreboardAssists   uint64
	ScoreboardDeaths    uint64
	ScoreboardHealing   uint64
	Suicides            uint64
	Captures            uint64
	CapturesBlocked     uint64
	ScoreboardDamage    uint64
	Extinguishes        uint64
	Ignites             uint64
	ObjectsBuilt        uint64
	ObjectsDestroyed    uint64
	BuildingsBuilt      uint64
	BUildingsDestroyed  uint64

	Personaname string
	AvatarHash  string
}

func (s Stats) WeaponList(ctx context.Context) ([]string, error) {
	return s.repo.WeaponList(ctx)
}

func (s Stats) Buckets(ctx context.Context) ([]Bucket, error) {
	return s.repo.Buckets(ctx)
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
	if _, err := scheduler.NewJob(gocron.CronJob("5 0 * * *", false), gocron.NewTask(s.UpdateDailyViews(ctx))); err != nil {
		return errors.Join(err, ErrJob)
	}
	// Sunday 00:00
	// scheduler.NewJob(gocron.CronJob("0 0 * * 0", false), gocron.NewTask(s.updateDailyViews(ctx)))
	// 1st of month 00:00
	// scheduler.NewJob(gocron.CronJob("0 0 1 * *", false), gocron.NewTask(s.updateDailyViews(ctx)))

	scheduler.Start()

	<-ctx.Done()

	if errShutdown := scheduler.Shutdown(); errShutdown != nil {
		return errors.Join(errShutdown, ErrJob)
	}

	return nil
}

func (s Stats) UpdateDailyViews(ctx context.Context) func() {
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

func (s Stats) MatchesWithPlayer(ctx context.Context, steamID steamid.SteamID) ([]PlayerMatchHistory, error) {
	return s.repo.MatchesWithPlayer(ctx, steamID)
}

type MatchesOpts struct {
	query.Filter

	serverID      uint32
	statsBucketID uint32
	mapID         int32
}

func (s Stats) Matches(ctx context.Context, opts MatchesOpts) ([]Match, uint64, error) {
	return s.repo.Matches(ctx, opts)
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

func (s Stats) Query(ctx context.Context, opts Opts) ([]any, uint64, error) {
	if (opts.Variant == VariantWeapons || opts.Variant == VariantClasses) && opts.VariantKey == "" {
		return nil, 0, fmt.Errorf("%w: variantKey must be set ", rpc.ErrBadRequest)
	}

	return s.repo.Query(ctx, opts)
}
