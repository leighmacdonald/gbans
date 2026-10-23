package stats

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/database/query"
)

// MapStat is the per-map aggregate row returned by the /maps table. All count
// and duration metrics are additive; unique players are exact for the queried
// range and must not be summed across ranges.
type MapStat struct {
	MapID             int32
	MapName           string
	MatchesPlayed     uint64
	MatchesDecided    uint64
	MatchesDrawn      uint64
	RedWins           uint64
	BluWins           uint64
	TotalDurationMS   uint64
	RoundsPlayed      uint64
	StalemateRounds   uint64
	UniquePlayers     uint64
	PlayerAppearances uint64
	Kills             uint64
	Assists           uint64
	Deaths            uint64
	Damage            uint64
	Healing           uint64
	Captures          uint64
	CapturesBlocked   uint64
	FirstPlayedAt     time.Time
	LastPlayedAt      time.Time
}

// MapPrefixStat is the per-map-type-prefix aggregate for the /maps prefix
// chart. Workshop maps are counted under the prefix of their normalized name
// (workshop/pl_foo.ugc123 becomes pl_foo); maps without a prefix are counted
// under "other".
type MapPrefixStat struct {
	Prefix          string
	MapsPlayed      uint64
	MatchesPlayed   uint64
	MatchesDecided  uint64
	TotalDurationMS uint64
}

// MapStatsSummary is the set of totals shown in the /maps summary cards.
type MapStatsSummary struct {
	MapsPlayed      uint64
	MatchesPlayed   uint64
	MatchesDecided  uint64
	MatchesDrawn    uint64
	TotalDurationMS uint64
	RoundsPlayed    uint64
}

// MapStatsOpts selects a range of per-map statistics. A zero Start and End
// selects the all-time view; any non-zero bound falls back to the daily view
// so arbitrary rolling ranges can be queried.
type MapStatsOpts struct {
	query.Filter

	StatsBucketID uint32
	Start         time.Time
	End           time.Time
	Search        string
}

func (o MapStatsOpts) allTime() bool {
	return o.Start.IsZero() && o.End.IsZero()
}

// MapStatsHistoryOpts selects the bucketed time series for the /maps chart.
type MapStatsHistoryOpts struct {
	StatsBucketID uint32
	Start         time.Time
	End           time.Time
	TimeBucket    TimeBucket
	MapIDs        []int32
	TopN          uint32
}

// MapStatsPoint is one bucketed value in a MapStatsSeries.
type MapStatsPoint struct {
	DateBucket     time.Time
	MatchesPlayed  uint64
	MatchesDecided uint64
	DurationMS     uint64
}

// MapStatsSeries is one map's (or the combined "Other" series') bucketed
// values. The "Other" series always has MapID 0.
type MapStatsSeries struct {
	MapID   int32
	MapName string
	Points  []MapStatsPoint
}

func (s Stats) MapStats(ctx context.Context, opts MapStatsOpts) ([]MapStat, uint64, MapStatsSummary, error) {
	return s.repo.MapStats(ctx, opts)
}

func (s Stats) MapStatsTop(ctx context.Context, opts MapStatsOpts, limit uint64) ([]MapStat, error) {
	return s.repo.MapStatsTop(ctx, opts, limit)
}

func (s Stats) MapStatsPrefixes(ctx context.Context, opts MapStatsOpts) ([]MapPrefixStat, error) {
	return s.repo.MapStatsPrefixes(ctx, opts)
}

func (s Stats) MapStatsHistory(ctx context.Context, opts MapStatsHistoryOpts) ([]MapStatsSeries, error) {
	return s.repo.MapStatsHistory(ctx, opts)
}
