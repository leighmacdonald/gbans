package config

import (
	"github.com/leighmacdonald/gbans/internal/domain"
)

type configUsecase struct {
	configRepo domain.ConfigRepository
}

func NewConfigUsecase(conf domain.ConfigRepository) domain.ConfigUsecase {
	SetDefaultConfigValues()

	return &configUsecase{configRepo: conf}
}

func (c *configUsecase) ExtURL(obj domain.LinkablePath) string {
	return c.configRepo.Config().ExtURL(obj)
}

func (c *configUsecase) ExtURLRaw(path string, args ...any) string {
	return c.configRepo.Config().ExtURLRaw(path, args...)
}

func (c *configUsecase) Config() domain.Config {
	return c.configRepo.Config()
}

// Read reads in config file and ENV variables if set.
func (c *configUsecase) Read(noFileOk bool) error {
	return c.configRepo.Read(noFileOk)
}
