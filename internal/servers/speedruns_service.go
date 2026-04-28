package servers

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/maps"
	mapsv1 "github.com/leighmacdonald/gbans/internal/maps/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SpeedrunsService struct {
	serversv1connect.UnimplementedSpeedrunsServiceHandler

	speedruns Speedruns
}

func NewSpeedrunsService(speedruns Speedruns, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := serversv1connect.NewSpeedrunsServiceHandler(&SpeedrunsService{speedruns: speedruns}, option...)

	authMiddleware.UserRoute(serversv1connect.SpeedrunsServiceMapSpeedrunsProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(serversv1connect.SpeedrunsServiceOverallTopNProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(serversv1connect.SpeedrunsServiceOverallRecentProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(serversv1connect.SpeedrunsServiceSpeedrunCreateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(serversv1connect.SpeedrunsServiceQueryProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s SpeedrunsService) MapSpeedruns(ctx context.Context, req *v1.MapSpeedrunsRequest) (*v1.MapSpeedrunsResponse, error) {
	runs, errRuns := s.speedruns.ByMap(ctx, req.GetMapName())
	if errRuns != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.MapSpeedrunsResponse{Speedruns: make([]*v1.SpeedrunOverview, len(runs))}
	for idx, run := range runs {
		resp.Speedruns[idx] = toSpeedrunOverview(run)
	}

	return &resp, nil
}

func (s SpeedrunsService) OverallTopN(ctx context.Context, req *v1.OverallTopNRequest) (*v1.OverallTopNResponse, error) {
	top, errTop := s.speedruns.TopNOverall(ctx, req.GetCount())
	if errTop != nil {
		if errors.Is(errTop, ErrValueOutOfRange) {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.OverallTopNResponse{Speedruns: make(map[string]*v1.Speedruns)}
	for name, runs := range top {
		resp.Speedruns[name] = &v1.Speedruns{Speedruns: make([]*v1.Speedrun, len(runs))}
		for idx, run := range runs {
			resp.Speedruns[name].Speedruns[idx] = toSpeedrun(run)
		}
	}

	return &resp, nil
}

func (s SpeedrunsService) OverallRecent(ctx context.Context, req *v1.OverallRecentRequest) (*v1.OverallRecentResponse, error) {
	top, errTop := s.speedruns.Recent(ctx, req.GetCount())
	if errTop != nil {
		if errors.Is(errTop, ErrValueOutOfRange) {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.OverallRecentResponse{Speedruns: make([]*v1.SpeedrunMapOverview, len(top))}
	// for idx, run := range top {
	//	 resp.Speedruns[idx] = toSpeedrunOverview(run)
	// }
	return &resp, nil
}

func (s SpeedrunsService) SpeedrunCreate(_ context.Context, _ *v1.SpeedrunCreateRequest) (*v1.SpeedrunCreateResponse, error) {
	// newSpeedrun := req.GetSpeedrun()
	// speedrun, errSpeedrun := s.speedruns.Save(ctx, toSpeedrun())
	// if errSpeedrun != nil {
	// 	return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	// }

	return &v1.SpeedrunCreateResponse{Speedrun: nil}, nil
}

func (s SpeedrunsService) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	speedrun, errSpeedrun := s.speedruns.ByID(ctx, req.GetSpeedrunId())
	if errSpeedrun != nil {
		if errors.Is(errSpeedrun, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.QueryResponse{Speedrun: toSpeedrun(speedrun)}, nil
}

func toSpeedrunOverview(speedrun SpeedrunMapOverview) *v1.SpeedrunOverview {
	return &v1.SpeedrunOverview{
		SpeedrunId:   &speedrun.SpeedrunID,
		ServerId:     &speedrun.ServerID,
		Rank:         &speedrun.Rank,
		InitialRank:  &speedrun.InitialRank,
		Map:          toMap(speedrun.MapDetail),
		Duration:     durationpb.New(speedrun.Duration),
		PlayerCount:  &speedrun.PlayerCount,
		BotCount:     &speedrun.BotCount,
		CreatedOn:    timestamppb.New(speedrun.CreatedOn),
		Category:     ptr.To(string(speedrun.Category)),
		TotalPlayers: &speedrun.TotalPlayers,
	}
}

func toMap(mapDetail maps.Map) *mapsv1.Map {
	return &mapsv1.Map{
		MapId:     &mapDetail.MapID,
		Name:      &mapDetail.MapName,
		CreatedOn: timestamppb.New(mapDetail.CreatedOn),
		UpdatedOn: timestamppb.New(mapDetail.UpdatedOn),
	}
}

func toSpeedrun(speedrun Speedrun) *v1.Speedrun {
	resp := v1.Speedrun{
		Captures:     make([]*v1.Capture, len(speedrun.PointCaptures)),
		Participants: make([]*v1.Participant, len(speedrun.Players)),
		SpeedrunId:   &speedrun.SpeedrunID,
		ServerId:     &speedrun.ServerID,
		Rank:         &speedrun.Rank,
		InitialRank:  &speedrun.InitialRank,
		Map:          toMap(speedrun.MapDetail),
		Duration:     durationpb.New(speedrun.Duration),
		PlayerCount:  &speedrun.PlayerCount,
		BotCount:     &speedrun.BotCount,
		CreatedOn:    timestamppb.New(speedrun.CreatedOn),
		Category:     ptr.To(string(speedrun.Category)),
		TotalPlayers: &speedrun.TotalPlayers,
	}

	return &resp
}
