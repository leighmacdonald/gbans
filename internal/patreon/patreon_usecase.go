package patreon

import (
	"github.com/leighmacdonald/gbans/internal/domain"
	libpatreon "gopkg.in/mxpv/patreon-go.v1"
)

type patreonUsecase struct {
	pr      domain.PatreonRepository
	manager *Mananger
}

func NewPatreonUsecase(pr domain.PatreonRepository) domain.PatreonUsecase {
	return &patreonUsecase{pr: pr}
}

func (p patreonUsecase) Tiers() ([]libpatreon.Campaign, error) {
	return p.manager.Tiers()
}

func (p patreonUsecase) Pledges() ([]libpatreon.Pledge, map[string]*libpatreon.User, error) {
	return p.manager.Pledges()
}
