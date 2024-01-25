package usecase

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type configUsecase struct {
	configRepo domain.ConfigRepository
}

func NewConfigUsecase(conf domain.ConfigRepository) domain.ConfigUsecase {
	return &configUsecase{configRepo: conf}
}

func (c *configUsecase) Config() *config.Config {
	return c.configRepo.Config()
}
