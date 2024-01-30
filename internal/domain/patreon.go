package domain

import (
	"context"

	"gopkg.in/mxpv/patreon-go.v1"
)

type PatreonUsecase interface {
	Start(ctx context.Context)
	Tiers() ([]patreon.Campaign, error)
	Pledges() ([]patreon.Pledge, map[string]*patreon.User, error)
}

type PatreonRepository interface {
	SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error
	GetPatreonAuth(ctx context.Context) (string, string, error)
}
