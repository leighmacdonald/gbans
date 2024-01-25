package repository

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type configRepository struct {
	Conf *config.Config
}

func NewConfigRepository(conf *config.Config) domain.ConfigRepository {
	return &configRepository{Conf: conf}
}

func (c *configRepository) Config() *config.Config {
	return c.Conf
}
