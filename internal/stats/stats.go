package stats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	MinPlayers = 6
	MinDuraion = 300
	// MinKills is the minimum number of kills a player must have to appear on
	// any leaderboard other than healing. This keeps low-sample-size players
	// off the boards.
	MinKills = 5
	// MinHealing is the minimum amount of healing a player must have to appear
	// on the healing leaderboard.
	MinHealing = 1000
	// MaxMatchDuration caps the demo header duration (seconds) accepted for
	// match creation. Longer demos are idle/recordings, not real matches.
	MaxMatchDuration = 2 * time.Hour
)

var (
	ErrInvalidState     = errors.New("invalid demo state")
	ErrInvalidBucket    = errors.New("invalid stat bucket")
	ErrJob              = errors.New("stat update job error")
	ErrMatchFiltered    = errors.New("match filtered")
	ErrNotEnoughPlayers = errors.New("not enough players")
	ErrMatchTooLong     = errors.New("match duration too long")
)

type Stats struct {
	repo Repository
	maps maps.Maps

	notifier          notification.Notifier
	matchLogChannelID string
}

// Option configures optional Stats dependencies.
type Option func(*Stats)

// WithMatchLogNotifications enables posting a summary to Discord after every
// successfully imported match.
func WithMatchLogNotifications(notifier notification.Notifier, channelID string) Option {
	return func(s *Stats) {
		s.notifier = notifier
		s.matchLogChannelID = channelID
	}
}

func (s Stats) Delete(ctx context.Context, demoID int32) error {
	return s.repo.Delete(ctx, demoID)
}

func (s Stats) MatchIDByDemoID(ctx context.Context, demoID int32) (uuid.UUID, error) {
	return s.repo.MatchIDByDemoID(ctx, demoID)
}

type VariantStats struct {
	Rank    uint64
	SteamID steamid.SteamID
	Variant string

	Personaname         string
	AvatarHash          string
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
	Heals               uint64
	Healed              uint64
	CrossbowHeals       uint64
	CrossbowHealing     uint64
	HealOnHit           uint64
	Extinguishes        uint64
	BuildingHealing     uint64
	DroppedUbers        uint64
	Reflects            uint64
	Defenses            uint64
	DirectHits          uint64
	Teleports           uint64
	PushDistance        uint64
	EnvironmentalDeaths uint64
	EnvironmentalKills  uint64
	ObjectPlaced        uint64
	ObjectUpgraded      uint64
	ObjectCarried       uint64
	ObjectDropped       uint64
	ObjectRemoved       uint64
	ObjectDetonated     uint64
	AmmoPacks           uint64
	HealthPacks         uint64
	HealthPackHealing   uint64
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
	Heals               uint64
	Healed              uint64
	CrossbowHeals       uint64
	CrossbowHealing     uint64
	HealOnHit           uint64
	BuildingHealing     uint64
	DroppedUbers        uint64
	Reflects            uint64
	Defenses            uint64
	DirectHits          uint64
	Teleports           uint64
	PushDistance        uint64
	EnvironmentalDeaths uint64
	EnvironmentalKills  uint64
	ObjectPlaced        uint64
	ObjectUpgraded      uint64
	ObjectCarried       uint64
	ObjectDropped       uint64
	ObjectRemoved       uint64
	ObjectDetonated     uint64
	AmmoPacks           uint64
	HealthPacks         uint64
	HealthPackHealing   uint64

	Personaname string
	AvatarHash  string
}

func (s Stats) WeaponList(ctx context.Context) ([]string, error) {
	return s.repo.WeaponList(ctx)
}

func (s Stats) Buckets(ctx context.Context) ([]Bucket, error) {
	return s.repo.Buckets(ctx)
}

func New(repo Repository, maps maps.Maps, options ...Option) Stats {
	stats := Stats{repo: repo, maps: maps}

	for _, option := range options {
		option(&stats)
	}

	return stats
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

// StatViews lists every materialized view backing QueryStats, refreshed by
// the daily job. Keep in sync with database.validViewNames (minus
// stats_weapons_view, which is only refreshed here for convenience).
func StatViews() []string {
	return []string{
		"stats_summary_daily_overall_view",
		"stats_summary_daily_variants_view",
		"stats_summary_weekly_overall_view",
		"stats_summary_weekly_variants_view",
		"stats_summary_monthly_overall_view",
		"stats_summary_monthly_variants_view",
		"stats_summary_yearly_overall_view",
		"stats_summary_yearly_variants_view",
		"stats_summary_alltime_overall_view",  //nolint:misspell // name must match the database view.
		"stats_summary_alltime_variants_view", //nolint:misspell // name must match the database view.
		"stats_map_daily_view",
		"stats_map_daily_player_view",
		"stats_map_alltime_view", //nolint:misspell // name must match the database view.
		"stats_weapons_view",
	}
}

func (s Stats) UpdateDailyViews(ctx context.Context) func() {
	return func() {
		if err := s.RefreshViews(ctx); err != nil {
			slog.Error("Failed to refresh views", slog.String("error", err.Error()))
		}
	}
}

// viewRefreshTimeout bounds a full refresh of every stats summary view.
// Refreshing all views, including the all-time and yearly summaries, over a
// large stats database can take many minutes, so this is deliberately
// generous.
const viewRefreshTimeout = 30 * time.Minute

// RefreshViews refreshes every stats summary view, running the refreshes
// concurrently with at most a few in flight at once. Used by the daily
// scheduler and the admin RefreshStats RPC.
func (s Stats) RefreshViews(ctx context.Context) error {
	// Refreshing every view is long-running and must not be aborted if the
	// caller's context is canceled (e.g. the RefreshStats RPC request context,
	// which a client or proxy cancels when it stops waiting). Detach from that
	// cancellation and bound the work with our own generous timeout so the
	// views are always left fully refreshed.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), viewRefreshTimeout)
	defer cancel()

	// Refresh views in parallel, capped well below the core count to leave
	// headroom for the rest of the server (and because the DB pool, not the
	// CPU, is what ultimately limits how many refreshes can run at once).
	workers := max(runtime.NumCPU()-2, 1)

	sem := make(chan struct{}, workers)
	var (
		waitGroup sync.WaitGroup
		errMu     sync.Mutex
		firstErr  error
	)

	for _, viewName := range StatViews() {
		waitGroup.Add(1)
		sem <- struct{}{}
		go func(viewName string) {
			defer waitGroup.Done()
			defer func() { <-sem }()

			start := time.Now()
			if err := s.repo.RefreshMaterializedView(ctx, viewName); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("refresh view %s: %w", viewName, err)
				}
				errMu.Unlock()

				return
			}

			slog.Debug("Refreshed stats view", slog.String("view", viewName), slog.Duration("duration", time.Since(start)))
		}(viewName)
	}

	waitGroup.Wait()

	return firstErr
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

	serverIDs      []int32
	statsBucketID  uint32
	mapID          int32
	startTimeStart time.Time
	startTimeEnd   time.Time
}

func (s Stats) Matches(ctx context.Context, opts MatchesOpts) ([]Match, uint64, error) {
	return s.repo.Matches(ctx, opts)
}

// ValidPlayerCount returns the number of distinct real player SteamIDs
// referenced across all demo rounds, excluding BOTs, HLTV/SourceTV, replays
// and other fake players.
func ValidPlayerCount(demo *demoparse.Demo) int {
	seen := map[steamid.SteamID]struct{}{}

	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			if player.IsFakePlayer || player.IsHlTv || player.IsReplay {
				continue
			}

			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}

			seen[sid] = struct{}{}
		}
	}

	return len(seen)
}

// ValidateDemo rejects demos that should never produce a match: fewer than
// MinPlayers distinct real players, or a header duration longer than
// MaxMatchDuration. It performs no database access so demo ingestion can call
// it before persisting anything.
func ValidateDemo(demo *demoparse.Demo) error {
	if numPlayers := ValidPlayerCount(demo); numPlayers < MinPlayers {
		return fmt.Errorf("%w: %w: got %d players, need at least %d",
			ErrMatchFiltered, ErrNotEnoughPlayers, numPlayers, MinPlayers)
	}

	if duration := time.Duration(demo.Duration * float64(time.Second)); duration > MaxMatchDuration {
		return fmt.Errorf("%w: %w: got %s, max %s",
			ErrMatchFiltered, ErrMatchTooLong, duration, MaxMatchDuration)
	}

	return nil
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

	if len(demo.Rounds) == 0 {
		return nil, fmt.Errorf("%w not enough rounds", ErrInvalidState)
	}

	if err := ValidateDemo(demo); err != nil {
		return nil, err
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

	if s.notifier != nil && s.matchLogChannelID != "" {
		s.notifier.Send(notification.NewDiscord(s.matchLogChannelID, MatchMessage(*match)))
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

func (s Stats) MyRank(ctx context.Context, opts Opts, steamID steamid.SteamID) (uint64, error) {
	return s.repo.MyRank(ctx, opts, steamID)
}

func (s Stats) MyIndex(ctx context.Context, opts Opts, steamID steamid.SteamID, rank uint64) (uint64, error) {
	return s.repo.MyIndex(ctx, opts, steamID, rank)
}

// PlayerKillsAndRank returns steamID's kills and their rank (per the variant's
// rank column) for the window described by opts. The rank is 0 when the player
// does not meet the leaderboard's minimum sample size.
func (s Stats) PlayerKillsAndRank(ctx context.Context, opts Opts, steamID steamid.SteamID) (uint64, uint64, error) {
	return s.repo.PlayerKillsAndRank(ctx, opts, steamID)
}
