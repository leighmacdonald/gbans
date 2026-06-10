package stats

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	mapsv1 "github.com/leighmacdonald/gbans/internal/maps/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	v1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	"github.com/leighmacdonald/gbans/internal/stats/v1/statsv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// statsv1connect.UnimplementedStatsServiceHandler

	stats   Stats
	servers servers.Servers
}

func NewService(stats Stats, servers servers.Servers, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := statsv1connect.NewStatsServiceHandler(Service{stats: stats, servers: servers}, option...)

	authMiddleware.UserRoute(statsv1connect.StatsServiceMatchProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
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
		Rounds:          []*v1.Round{},
	}

	assembleRounds(out, match)
	assemblePlayers(out, match)
	assembleClasses(out, match)
	assembleWeapons(out, match)

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
					Weapons:             []*v1.RoundPlayerWeapon{},
					Classes:             []*v1.RoundPlayerClass{},
				})

				break
			}
		}
	}
}

func assembleWeapons(out *v1.Match, match *Match) {
	for _, cls := range match.Weapons {
		for _, round := range out.GetRounds() {
			if cls.RoundID != round.GetRoundId() {
				continue
			}

			for _, player := range round.GetPlayers() {
				if cls.SteamID.Int64() != player.GetSteamId() {
					continue
				}

				player.Weapons = append(player.Weapons, &v1.RoundPlayerWeapon{
					Weapon:              &cls.Weapon,
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

func assembleClasses(out *v1.Match, match *Match) {
	for _, cls := range match.Classes {
		for _, round := range out.GetRounds() {
			if cls.RoundID != round.GetRoundId() {
				continue
			}
			for _, player := range round.GetPlayers() {
				if cls.SteamID.Int64() != player.GetSteamId() {
					continue
				}
				player.Classes = append(player.Classes, &v1.RoundPlayerClass{
					Class:               &cls.Class,
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
