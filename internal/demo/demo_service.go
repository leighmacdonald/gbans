package demo

import (
	"context"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/demo/v1"
	"github.com/leighmacdonald/gbans/internal/demo/v1/demov1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DemoService struct {
	demov1connect.UnimplementedDemoServiceHandler

	demos Demos
}

func NewDemoService(demos Demos, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := demov1connect.NewDemoServiceHandler(&DemoService{demos: demos}, option...)

	authMiddleware.UserRoute(demov1connect.DemoServiceGetDemosProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(demov1connect.DemoServiceRunCleanupProcedure, rpc.WithMinPermissions(permission.Admin))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s DemoService) GetDemos(ctx context.Context, _ *emptypb.Empty) (*v1.GetDemosResponse, error) {
	demos, errDemos := s.demos.GetDemos(ctx)
	if errDemos != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GetDemosResponse{Demos: make([]*v1.Demo, len(demos))}
	for idx, demo := range demos {
		resp.Demos[idx] = &v1.Demo{
			DemoId:          &demo.DemoID,
			ServerId:        &demo.ServerID,
			ServerNameShort: &demo.ServerNameShort,
			ServerNameLong:  &demo.ServerNameLong,
			Title:           &demo.Title,
			CreatedOn:       timestamppb.New(demo.CreatedOn),
			Downloads:       &demo.Downloads,
			Size:            &demo.Size,
			MapName:         &demo.MapName,
			Archive:         &demo.Archive,
			Stats:           make(map[string]string),
			AssetId:         new(demo.AssetID.String()),
		}
		for k := range demo.Stats {
			resp.Demos[idx].Stats[k] = k
		}
	}

	return &resp, nil
}

func (s DemoService) RunCleanup(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.demos.Cleanup(ctx)

	return &emptypb.Empty{}, nil
}
