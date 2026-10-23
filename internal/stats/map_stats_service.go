package stats

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s Service) QueryMapStats(ctx context.Context, request *v1.QueryMapStatsRequest) (*v1.QueryMapStatsResponse, error) {
	opts := MapStatsOpts{
		Filter:        rpc.FromRPC(request.GetFilter()),
		StatsBucketID: request.GetStatsBucketId(),
		Search:        request.GetSearch(),
	}
	if start := request.GetStart(); start != nil {
		opts.Start = start.AsTime()
	}
	if end := request.GetEnd(); end != nil {
		opts.End = end.AsTime()
	}
	if !opts.Start.IsZero() && !opts.End.IsZero() && opts.End.Before(opts.Start) {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	rows, count, summary, errRows := s.stats.MapStats(ctx, opts)
	if errRows != nil {
		slog.Error("Failed to query map stats",
			slog.Uint64("statsBucketID", uint64(opts.StatsBucketID)),
			slog.String("error", errRows.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	top, errTop := s.stats.MapStatsTop(ctx, opts, 10)
	if errTop != nil {
		slog.Error("Failed to query top map stats",
			slog.Uint64("statsBucketID", uint64(opts.StatsBucketID)),
			slog.String("error", errTop.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &v1.QueryMapStatsResponse{
		Count:   &count,
		Summary: toMapStatsSummary(summary),
		Maps:    make([]*v1.MapStat, len(rows)),
		TopMaps: make([]*v1.MapStat, len(top)),
	}
	for idx := range rows {
		resp.Maps[idx] = toMapStat(rows[idx])
	}
	for idx := range top {
		resp.TopMaps[idx] = toMapStat(top[idx])
	}

	return resp, nil
}

func (s Service) QueryMapStatsHistory(ctx context.Context, request *v1.QueryMapStatsHistoryRequest) (*v1.QueryMapStatsHistoryResponse, error) {
	timeBucket := TimeBucket(request.GetTimeBucket())
	if _, ok := mapStatsUnit(timeBucket); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	opts := MapStatsHistoryOpts{
		StatsBucketID: request.GetStatsBucketId(),
		TimeBucket:    timeBucket,
		TopN:          request.GetTopN(),
		MapIDs:        request.GetMapIds(),
	}
	if start := request.GetStart(); start != nil {
		opts.Start = start.AsTime()
	}
	if end := request.GetEnd(); end != nil {
		opts.End = end.AsTime()
	}
	if !opts.Start.IsZero() && !opts.End.IsZero() && opts.End.Before(opts.Start) {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	series, errSeries := s.stats.MapStatsHistory(ctx, opts)
	if errSeries != nil {
		if errors.Is(errSeries, ErrInvalidMapStatsBucket) {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}
		slog.Error("Failed to query map stats history",
			slog.Uint64("statsBucketID", uint64(opts.StatsBucketID)),
			slog.String("error", errSeries.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &v1.QueryMapStatsHistoryResponse{Series: make([]*v1.MapStatsSeries, len(series))}
	for idx := range series {
		resp.Series[idx] = toMapStatsSeries(series[idx])
	}

	return resp, nil
}

func toMapStat(stat MapStat) *v1.MapStat {
	return &v1.MapStat{
		MapId:             &stat.MapID,
		MapName:           &stat.MapName,
		MatchesPlayed:     &stat.MatchesPlayed,
		MatchesDecided:    &stat.MatchesDecided,
		MatchesDrawn:      &stat.MatchesDrawn,
		RedWins:           &stat.RedWins,
		BluWins:           &stat.BluWins,
		TotalDurationMs:   &stat.TotalDurationMS,
		RoundsPlayed:      &stat.RoundsPlayed,
		StalemateRounds:   &stat.StalemateRounds,
		UniquePlayers:     &stat.UniquePlayers,
		PlayerAppearances: &stat.PlayerAppearances,
		Kills:             &stat.Kills,
		Assists:           &stat.Assists,
		Deaths:            &stat.Deaths,
		Damage:            &stat.Damage,
		Healing:           &stat.Healing,
		Captures:          &stat.Captures,
		CapturesBlocked:   &stat.CapturesBlocked,
		FirstPlayedAt:     timestamppb.New(stat.FirstPlayedAt),
		LastPlayedAt:      timestamppb.New(stat.LastPlayedAt),
	}
}

func toMapStatsSummary(summary MapStatsSummary) *v1.MapStatsSummary {
	return &v1.MapStatsSummary{
		MapsPlayed:      &summary.MapsPlayed,
		MatchesPlayed:   &summary.MatchesPlayed,
		MatchesDecided:  &summary.MatchesDecided,
		MatchesDrawn:    &summary.MatchesDrawn,
		TotalDurationMs: &summary.TotalDurationMS,
		RoundsPlayed:    &summary.RoundsPlayed,
	}
}

func toMapStatsSeries(series MapStatsSeries) *v1.MapStatsSeries {
	points := make([]*v1.MapStatsPoint, len(series.Points))
	for idx := range series.Points {
		points[idx] = &v1.MapStatsPoint{
			DateBucket:     timestamppb.New(series.Points[idx].DateBucket),
			MatchesPlayed:  &series.Points[idx].MatchesPlayed,
			MatchesDecided: &series.Points[idx].MatchesDecided,
			DurationMs:     &series.Points[idx].DurationMS,
		}
	}

	return &v1.MapStatsSeries{
		MapId:   &series.MapID,
		MapName: &series.MapName,
		Points:  points,
	}
}
