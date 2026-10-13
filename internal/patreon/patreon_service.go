package patreon

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/patreon/v1"
	"github.com/leighmacdonald/gbans/internal/patreon/v1/patreonv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	patreon Patreon
}

func NewService(patreon Patreon, authMiddleware *rpc.Middleware, options ...connect.HandlerOption) rpc.Service {
	pattern, handler := patreonv1connect.NewPatreonServiceHandler(Service{patreon: patreon}, options...)

	authMiddleware.PublicRoute(patreonv1connect.PatreonServicePatronCampaignsProcedure)
	authMiddleware.AuthedRoute(patreonv1connect.PatreonServicePatronLoginProcedure)
	authMiddleware.AuthedRoute(patreonv1connect.PatreonServicePatronLogoutProcedure)

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) PatronCampaigns(_ context.Context, _ *emptypb.Empty) (*v1.PatronCampaignsResponse, error) {
	campaign := s.patreon.Campaign()
	if campaign.ID == "" {
		return &v1.PatronCampaignsResponse{Campaigns: []*v1.Campaign{}}, nil
	}

	return &v1.PatronCampaignsResponse{Campaigns: []*v1.Campaign{toCampaign(campaign)}}, nil
}

func (s Service) PatronLogin(ctx context.Context, _ *emptypb.Empty) (*v1.PatronLoginResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)

	return &v1.PatronLoginResponse{Url: new(s.patreon.CreateOAuthRedirect(user.GetSteamID()))}, nil
}

func (s Service) PatronLogout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)

	if err := s.patreon.Forget(ctx, user.GetSteamID()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func nullTimeToProto(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}

	return timestamppb.New(value)
}

func toCampaign(campaign Campaign) *v1.Campaign {
	return &v1.Campaign{
		Type: new("campaign"),
		Id:   new(campaign.ID),
		Attributes: &v1.Attributes{
			CreatedAt:     nullTimeToProto(campaign.CreatedAt),
			CreationName:  new(campaign.CreationName),
			ImageSmallUrl: new(campaign.ImageSmallURL),
			ImageUrl:      new(campaign.ImageURL),
			IsMonthly:     new(campaign.IsMonthly),
			PatronCount:   new(campaign.PatronCount),
			PublishedAt:   nullTimeToProto(campaign.PublishedAt),
			Summary:       new(campaign.Summary),
			ThanksMsg:     new(campaign.ThanksMsg),
			Url:           new(campaign.URL),
			Vanity:        new(campaign.Vanity),
		},
	}
}
