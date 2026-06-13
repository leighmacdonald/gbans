package stats

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	mapsv1 "github.com/leighmacdonald/gbans/internal/maps/v1"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	v1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	"github.com/leighmacdonald/gbans/internal/stats/v1/statsv1connect"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errMatchesWithPlayerNotImplemented = errors.New("stats.v1.StatsService.MatchesWithPlayer is not implemented")

type Service struct {
	// statsv1connect.UnimplementedStatsServiceHandler

	stats   Stats
	servers *servers.Servers
}

func NewService(stats Stats, servers *servers.Servers, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := statsv1connect.NewStatsServiceHandler(Service{stats: stats, servers: servers}, option...)

	authMiddleware.UserRoute(statsv1connect.StatsServiceMatchProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(statsv1connect.StatsServiceBucketsProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(statsv1connect.StatsServiceQueryProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) MatchesWithPlayer(ctx context.Context, request *v1.MatchesWithPlayerRequest) (*v1.MatchesWithPlayerResponse, error) {
	matches, errMatches := s.stats.MatchesWithPlayer(ctx, steamid.New(request.GetSteamId()))
	if errMatches != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &v1.MatchesWithPlayerResponse{Matches: make([]*v1.PlayerMatchHistory, len(matches))}
	for idx, match := range matches {
		resp.Matches[idx] = &v1.PlayerMatchHistory{
			MatchId:         new(match.MatchID.String()),
			ServerId:        &match.ServerID,
			ServerName:      &match.ServerName,
			ServerNameShort: &match.ServerNameShort,
			DemoId:          &match.DemoID,
			MapId:           &match.MapID,
			MapName:         &match.MapName,
			BucketId:        &match.BucketID,
			BucketName:      &match.BucketName,
			Hostname:        &match.Hostname,
			ScoreRed:        &match.ScoreRed,
			ScoreBlu:        &match.ScoreBlu,
			Duration:        &match.DurationMs,
			CreatedOn:       timestamppb.New(match.CreatedOn),
		}
	}

	return resp, nil
}

func (s Service) WeaponList(ctx context.Context, _ *emptypb.Empty) (*v1.WeaponListResponse, error) {
	weapons, errWeapons := s.stats.WeaponList(ctx)
	if errWeapons != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.WeaponListResponse{Weapons: weapons}, nil
}

func (s Service) Buckets(ctx context.Context, _ *emptypb.Empty) (*v1.BucketsResponse, error) {
	buckets, errBuckets := s.stats.Buckets(ctx)
	if errBuckets != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	resp := &v1.BucketsResponse{Buckets: make([]*v1.Bucket, len(buckets))}
	for idx := range buckets {
		resp.Buckets[idx] = &v1.Bucket{
			StatsBucketId: &buckets[idx].BucketID,
			BucketName:    &buckets[idx].BucketName,
		}
	}

	return resp, nil
}

func createResp(opts Opts, request *v1.QueryRequest, count uint64, size int) *v1.QueryResponse {
	resp := &v1.QueryResponse{
		Variant: request.Variant,
		Count:   &count,
	}
	switch opts.Variant {
	case VariantWeapons:
		fallthrough
	case VariantClasses:
		resp.StatContainer = &v1.QueryResponse_StatsVariant{
			StatsVariant: &v1.VariantStatsContainer{
				Stats: make([]*v1.VariantStats, size),
			},
		}
	}

	return resp
}

func (s Service) Query(ctx context.Context, request *v1.QueryRequest) (*v1.QueryResponse, error) {
	opts := Opts{
		Variant:    Variant(request.GetVariant()),
		VariantKey: request.GetVariantKey(),
		Filter:     rpc.FromRPC(request.GetFilter()),
		TimeBucket: TimeBucket(request.GetTimeBucket()),
		TimeStamp:  request.GetTime().AsTime(),
	}
	statsBucketID := request.GetStatsBucketId()

	stats, count, errStats := s.stats.Query(ctx, statsBucketID, opts)
	resp := createResp(opts, request, count, len(stats))
	if errStats != nil {
		if errors.Is(errStats, database.ErrNoResult) {
			return resp, nil
		}
		slog.Error("Failed to load stats",
			slog.String("variant", opts.Variant.String()),
			slog.Time("timeStamp", opts.TimeStamp),
			slog.String("timeBucket", opts.TimeBucket.String()),
			slog.Uint64("statsBucketID", uint64(statsBucketID)),
			slog.String("variantFilter", opts.VariantKey),
			slog.String("error", errStats.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	for idx, statRow := range stats {
		switch opts.Variant {
		case VariantClasses:
			fallthrough
		case VariantWeapons:
			variantStats, ok := statRow.(VariantStats)
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
			}
			container, ok := resp.GetStatContainer().(*v1.QueryResponse_StatsVariant)
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
			}
			container.StatsVariant.Stats[idx] = toVariantStats(variantStats)
		}
	}

	return resp, nil
}

func toVariantStats(stats VariantStats) *v1.VariantStats {
	return &v1.VariantStats{
		Variant: &stats.Variant,
		Rank:    &stats.Rank,
		Player: &personv1.PersonDisplay{
			SteamId: new(stats.SteamID.Int64()),
			Name:    new(stats.SteamID.String()),
		},
		Kills:               &stats.Kills,
		Assists:             &stats.Assists,
		Deaths:              &stats.Deaths,
		PostroundKills:      &stats.PostroundKills,
		PostroundAssists:    &stats.PostroundAssists,
		PostroundDeaths:     &stats.PostroundDeaths,
		Damage:              &stats.Damage,
		DamageTaken:         &stats.DamageTaken,
		Dominations:         &stats.Dominations,
		Dominated:           &stats.Dominated,
		Revenges:            &stats.Revenges,
		Revenged:            &stats.Revenged,
		Airshots:            &stats.Airshots,
		HeadshotKills:       &stats.HeadshotKills,
		BackstabKills:       &stats.BackstabKills,
		Headshots:           &stats.Headshots,
		Backstabs:           &stats.Backstabs,
		WasHeadshot:         &stats.WasHeadshot,
		WasBackstabbed:      &stats.WasBackstabbed,
		PreroundHealing:     &stats.PreroundHealing,
		Healing:             &stats.Healing,
		PostroundHealing:    &stats.PostroundHealing,
		Drops:               &stats.Drops,
		NearFullChargeDeath: &stats.NearFullChargeDeath,
		ChargesUber:         &stats.ChargesUber,
		ChargesKritz:        &stats.ChargesKritz,
		ChargesVacc:         &stats.ChargesVacc,
		ChargesQuickfix:     &stats.ChargesQuickfix,
	}
}

func (s Service) Match(ctx context.Context, request *v1.MatchRequest) (*v1.MatchResponse, error) {
	matchID, errMatchID := uuid.FromString(request.GetMatchId())
	if errMatchID != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	match, errMatch := s.loadMatch(ctx, matchID)
	if errMatch != nil {
		return nil, errMatch
	}

	return &v1.MatchResponse{Match: match}, nil
}

func (s Service) loadMatch(ctx context.Context, matchID uuid.UUID) (*v1.Match, error) {
	match, errMatch := s.stats.Match(ctx, matchID)
	if errMatch != nil {
		if errors.Is(errMatch, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	mapInfo, errMap := s.stats.maps.GetByID(ctx, match.MapID)
	if errMap != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	bucket, errBucket := s.stats.Bucket(ctx, match.StatsBucketID)
	if errBucket != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	server, errServer := s.servers.Server(ctx, match.ServerID)
	if errServer != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	out := &v1.Match{
		Overview: &v1.MatchOverview{
			MatchId:         new(matchID.String()),
			ServerId:        &match.ServerID,
			ServerName:      &server.Name,
			ServerNameShort: &server.ShortName,
			DemoId:          &match.DemoID,
			Map: &mapsv1.Map{
				MapId:     &mapInfo.MapID,
				Name:      &mapInfo.MapName,
				CreatedOn: timestamppb.New(mapInfo.CreatedOn),
				UpdatedOn: timestamppb.New(mapInfo.UpdatedOn),
			},
			StatsBucketId:   &bucket.BucketID,
			StatsBucketName: &bucket.BucketName,
			Hostname:        &match.Hostname,
			ScoreRed:        &match.ScoreRed,
			ScoreBlu:        &match.ScoreBlu,
			Duration:        &match.DurationMs,
			StartTime:       timestamppb.New(match.StartTime),
			CreatedOn:       timestamppb.New(match.CreatedOn),
		},
		Rounds: []*v1.Round{},
	}

	assembleRounds(out, match)
	assemblePlayers(out, match)
	assembleVariants(out, match)

	return out, nil
}

func assemblePlayers(out *v1.Match, match *Match) {
	for _, player := range match.Players {
		for _, round := range out.GetRounds() {
			if round.GetRoundId() == player.RoundID {
				round.Players = append(round.Players, &v1.RoundPlayer{
					RoundId:             &player.RoundID,
					SteamId:             new(player.SteamID.Int64()),
					Team:                toTeam(player.Team),
					Mvp:                 &player.MVP,
					TickStart:           &player.TickStart,
					TickEnd:             &player.TickEnd,
					Points:              &player.Points,
					ConnectionCount:     &player.ConnectionCount,
					BonusPoints:         &player.BonusPoints,
					Kills:               &player.Kills,
					Assists:             &player.Assists,
					Deaths:              &player.Deaths,
					PostroundKills:      &player.PostroundKills,
					PostroundAssists:    &player.PostroundAssists,
					PreroundHealing:     &player.PreroundHealing,
					Healing:             &player.Healing,
					Drops:               &player.Drops,
					NearFullChargeDeath: &player.NearFullChargeDeath,
					ChargesUber:         &player.ChargesUber,
					ChargesKritz:        &player.ChargesKritz,
					ChargesVacc:         &player.ChargesVacc,
					ChargesQuickfix:     &player.ChargesQuickfix,
					Damage:              &player.Damage,
					DamageTaken:         &player.DamageTaken,
					Dominations:         &player.Dominations,
					Dominated:           &player.Dominated,
					Revenges:            &player.Revenges,
					Revenged:            &player.Revenged,
					Airshots:            &player.Airshots,
					Headshots:           &player.Headshots,
					HeadshotKills:       &player.HeadshotKills,
					Backstabs:           &player.Backstabs,
					BackstabKills:       &player.BackstabKills,
					WasHeadshot:         &player.WasHeadshot,
					WasBackstabbed:      &player.WasBackstabbed,
					Shots:               &player.Shots,
					Hits:                &player.Hits,
					ObjectsBuilt:        &player.ObjectsBuilt,
					ObjectsDestroyed:    &player.ObjectsDestroyed,
					ScoreboardKills:     &player.ScoreboardKills,
					ScoreboardAssists:   &player.ScoreboardAssists,
					ScoreboardDeaths:    &player.ScoreboardDeaths,
					PostroundDeaths:     &player.PostroundDeaths,
					Captures:            &player.Captures,
					CapturesBlocked:     &player.CapturesBlocked,
					ScoreboardDamage:    &player.ScoreboardDamage,
					Extinguishes:        &player.Extinguishes,
					Ignites:             &player.Ignites,
					BuildingsBuilt:      &player.BuildingsBuilt,
					BuildingsDestroyed:  &player.BUildingsDestroyed,
					Variants:            []*v1.RoundPlayerVariant{},
				})

				break
			}
		}
	}
}

func assembleVariants(out *v1.Match, match *Match) {
	for _, cls := range match.Variants {
		for _, round := range out.GetRounds() {
			if cls.RoundID != round.GetRoundId() {
				continue
			}

			for _, player := range round.GetPlayers() {
				if cls.SteamID.Int64() != player.GetSteamId() {
					continue
				}

				player.Variants = append(player.Variants, &v1.RoundPlayerVariant{
					Variant:             &cls.Variant,
					RoundId:             &cls.RoundID,
					SteamId:             new(cls.SteamID.Int64()),
					Kills:               &cls.Kills,
					Assists:             &cls.Assists,
					Deaths:              &cls.Deaths,
					PostroundKills:      &cls.PostroundKills,
					PostroundAssists:    &cls.PostroundAssists,
					PostroundDeaths:     &cls.PostroundDeaths,
					Damage:              &cls.Damage,
					DamageTaken:         &cls.DamageTaken,
					Dominations:         &cls.Dominations,
					Dominated:           &cls.Dominated,
					Revenges:            &cls.Revenges,
					Revenged:            &cls.Revenged,
					Airshots:            &cls.Airshots,
					HeadshotKills:       &cls.HeadshotKills,
					BackstabKills:       &cls.BackstabKills,
					Headshots:           &cls.Headshots,
					Backstabs:           &cls.Backstabs,
					WasHeadshot:         &cls.WasHeadshot,
					WasBackstabbed:      &cls.WasBackstabbed,
					PreroundHealing:     &cls.PreroundHealing,
					Healing:             &cls.Healing,
					PostroundHealing:    &cls.PostroundHealing,
					Drops:               &cls.Drops,
					NearFullChargeDeath: &cls.NearFullChargeDeath,
					ChargesUber:         &cls.ChargesUber,
					ChargesKritz:        &cls.ChargesKritz,
					ChargesVacc:         &cls.ChargesVacc,
					ChargesQuickfix:     &cls.ChargesQuickfix,
				})

				break
			}
		}
	}
}

func assembleRounds(out *v1.Match, match *Match) {
	for _, round := range match.Rounds {
		rnd := &v1.Round{
			RoundId:       &round.RoundID,
			Winner:        toTeam(round.Winner),
			IsStalemate:   &round.IsStalemate,
			IsSuddenDeath: &round.IsStalemate,
			DurationMs:    &round.DurationMs,
			Players:       []*v1.RoundPlayer{},
		}

		out.Rounds = append(out.Rounds, rnd)
	}
}

func toTeam(team string) *v1.Team {
	switch team {
	case "blu":
		return new(v1.Team_TEAM_BLU)
	case "red":
		return new(v1.Team_TEAM_RED)
	case "spec":
		return new(v1.Team_TEAM_SPEC)
	case "unassigned":
		fallthrough
	case "":
		fallthrough
	default:
		return new(v1.Team_TEAM_UNASSIGNED_UNSPECIFIED)
	}
}
