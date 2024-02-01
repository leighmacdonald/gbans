package patreon

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
	"go.uber.org/zap"
	libpatreon "gopkg.in/mxpv/patreon-go.v1"
)

type patreonUsecase struct {
	repository domain.PatreonRepository
	manager    *Manager
}

func (p patreonUsecase) Start(ctx context.Context) {
	p.manager.Start(ctx)
}

func NewPatreonUsecase(logger *zap.Logger, repository domain.PatreonRepository) domain.PatreonUsecase {
	return &patreonUsecase{repository: repository, manager: NewPatreonManager(logger)}
}

func (p patreonUsecase) Tiers() ([]libpatreon.Campaign, error) {
	return p.manager.Tiers()
}

func (p patreonUsecase) Pledges() ([]libpatreon.Pledge, map[string]*libpatreon.User, error) {
	return p.manager.Pledges()
}
