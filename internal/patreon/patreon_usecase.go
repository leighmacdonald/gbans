package patreon

import (
	"context"
	"go.uber.org/zap"

	"github.com/leighmacdonald/gbans/internal/domain"
	libpatreon "gopkg.in/mxpv/patreon-go.v1"
)

type patreonUsecase struct {
	pr      domain.PatreonRepository
	manager *Manager
}

func (p patreonUsecase) Start(ctx context.Context) {
	p.manager.Start(ctx)
}

func NewPatreonUsecase(logger *zap.Logger, pr domain.PatreonRepository) domain.PatreonUsecase {
	return &patreonUsecase{pr: pr, manager: NewPatreonManager(logger)}
}

func (p patreonUsecase) Tiers() ([]libpatreon.Campaign, error) {
	return p.manager.Tiers()
}

func (p patreonUsecase) Pledges() ([]libpatreon.Pledge, map[string]*libpatreon.User, error) {
	return p.manager.Pledges()
}
