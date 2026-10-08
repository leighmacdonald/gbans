package demo

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/asset"
	v1 "github.com/leighmacdonald/gbans/internal/demo/v1"
	"github.com/leighmacdonald/gbans/internal/demo/v1/demov1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// demov1connect.UnimplementedDemoServiceHandler

	demos Demos
}

func NewService(demos Demos, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := demov1connect.NewDemoServiceHandler(&Service{demos: demos}, option...)

	authMiddleware.UserRoute(demov1connect.DemoServiceGetDemoProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_DEMO_READ))
	authMiddleware.UserRoute(demov1connect.DemoServiceGetDemosProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_DEMO_READ))
	authMiddleware.UserRoute(demov1connect.DemoServiceRunCleanupProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_DEMO_ADMIN))
	authMiddleware.UserRoute(demov1connect.DemoServiceUploadDemoProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_DEMO_ADMIN))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) GetDemo(ctx context.Context, req *v1.GetDemoRequest) (*v1.GetDemoResponse, error) {
	demo, errDemos := s.demos.GetDemoByID(ctx, req.GetDemoId())
	if errDemos != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &v1.GetDemoResponse{Demo: &v1.Demo{
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
	}}

	for k := range demo.Stats {
		resp.Demo.Stats[k] = k
	}

	return resp, nil
}

func (s Service) GetDemos(ctx context.Context, _ *emptypb.Empty) (*v1.GetDemosResponse, error) {
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

func (s Service) RunCleanup(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.demos.Cleanup(ctx)

	return &emptypb.Empty{}, nil
}

func (s Service) UploadDemo(ctx context.Context, req *v1.UploadDemoRequest) (*v1.UploadDemoResponse, error) {
	demoFile, matchID, errUpload := s.demos.Upload(ctx, UploadedDemo{
		Name:     req.GetFilename(),
		ServerID: req.GetServerId(),
		Content:  req.GetContents(),
	}, req.GetForce())
	if errUpload != nil {
		if errors.Is(errUpload, ErrDemoFilename) || errors.Is(errUpload, ErrServerValidate) ||
			errors.Is(errUpload, asset.ErrAssetTooLarge) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errUpload)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.UploadDemoResponse{
		DemoId:  new(demoFile.DemoID),
		AssetId: new(demoFile.AssetID.String()),
		MatchId: new(matchID.String()),
	}, nil
}
