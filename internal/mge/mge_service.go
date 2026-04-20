package mge

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	v1 "github.com/leighmacdonald/gbans/internal/mge/v1"
	"github.com/leighmacdonald/gbans/internal/mge/v1/mgev1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	mgev1connect.UnimplementedMGEServiceHandler

	mge MGE
}

func NewService(mge MGE, authMiddleware *rpc.Middleware, options ...connect.HandlerOption) rpc.Service {
	pattern, handler := mgev1connect.NewMGEServiceHandler(Service{mge: mge}, options...)

	authMiddleware.AuthedRoute(mgev1connect.MGEServiceGetRatingsOverallProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.AuthedRoute(mgev1connect.MGEServiceGetHistoryProcedure, rpc.WithMinPermissions(permission.Guest))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) GetRatingsOverall(ctx context.Context, req *v1.GetRatingsOverallRequest) (*v1.GetRatingsOverallResponse, error) {
	history, count, errChat := s.mge.Query(ctx, QueryOpts{
		Filter:  rpc.FromRPC(req.GetFilter()),
		SteamID: req.GetSteamId(),
	})
	if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GetRatingsOverallResponse{Stats: make([]*v1.PlayerStats, len(history)), Count: &count}
	for idx, hist := range history {
		resp.Stats[idx] = &v1.PlayerStats{
			StatsId:     &hist.StatsID,
			Rating:      &hist.Rating,
			SteamId:     ptr.To(hist.SteamID.Int64()),
			PersonaName: &hist.Personaname,
			AvatarHash:  &hist.Avatarhash,
			Name:        &hist.Name,
			Wins:        &hist.Wins,
			Losses:      &hist.Losses,
			LastPlayed:  timestamppb.New(hist.LastPlayed),
		}
	}

	return &resp, nil
}

func (s Service) GetHistory(ctx context.Context, req *v1.GetHistoryRequest) (*v1.GetHistoryResponse, error) {
	history, count, errChat := s.mge.History(ctx, HistoryOpts{
		Filter:  rpc.FromRPC(req.GetFilter()),
		Mode:    DuelMode(ptr.From(req.Mode)),
		Winner:  req.GetWinner(),
		Loser:   req.GetLoser(),
		Winner2: req.GetWinner2(),
		Loser2:  req.GetLoser2(),
	})
	if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GetHistoryResponse{History: make([]*v1.Duel, len(history)), Count: &count}
	for idx, hist := range history {
		resp.History[idx] = &v1.Duel{
			DuelId:             &hist.DuelID,
			Winner:             ptr.To(hist.Winner.Int64()),
			WinnerAvatarHash:   &hist.WinnerAvatarhash,
			WinnerPersonaName:  &hist.WinnerPersonaname,
			Winner2:            ptr.To(hist.Winner2.Int64()),
			Winner2AvatarHash:  &hist.Winner2Avatarhash,
			Winner2PersonaName: &hist.Winner2Personaname,
			Loser:              ptr.To(hist.Loser.Int64()),
			LoserAvatarHash:    &hist.LoserAvatarhash,
			LoserPersonaName:   &hist.LoserPersonaname,
			Loser2:             ptr.To(hist.Loser2.Int64()),
			Loser2AvatarHash:   &hist.Loser2Avatarhash,
			Loser2PersonaName:  &hist.Loser2Personaname,
			WinnerScore:        &hist.WinnerScore,
			LoserScore:         &hist.LoserScore,
			WinLimit:           &hist.Winlimit,
			GameTime:           timestamppb.New(hist.GameTime),
			MapName:            &hist.MapName,
			ArenaName:          &hist.ArenaName,
		}
	}

	return &resp, nil
}
