package servers

import (
	"context"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DemoService struct {
	serversv1connect.UnimplementedDemoServiceHandler

	demos Demos
}

func NewDemoService(demos Demos) *DemoService {
	return &DemoService{demos: demos}
}

func (s DemoService) GetDemos(ctx context.Context, _ *emptypb.Empty) (*v1.GetDemosResponse, error) {
	demos, errDemos := s.demos.GetDemos(ctx)
	if errDemos != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GetDemosResponse{Demos: make([]*v1.Demo, len(demos))}
	for idx, demo := range demos {
		var stats map[string]string
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
			Stats:           stats,
			AssetId:         ptr.To(demo.AssetID.String()),
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
