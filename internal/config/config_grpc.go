package config

import (
	"context"

	configv1 "github.com/leighmacdonald/gbans/internal/rpc/config/v1"
	"github.com/leighmacdonald/gbans/internal/rpc/config/v1/configv1connect"
)

type RPC struct {
	configv1connect.UnimplementedConfigServiceHandler
}

func (s *RPC) Info(ctx context.Context, req *configv1.InfoRequest) (*configv1.InfoResponse, error) {
	return &configv1.InfoResponse{}, nil
}
