package config

import (
	"github.com/leighmacdonald/gbans/internal/domain"
)

type configUsecase struct {
	configRepo domain.ConfigRepository
}

func NewConfigUsecase(repository domain.ConfigRepository) domain.ConfigUsecase {
	setDefaultConfigValues()

	return &configUsecase{configRepo: repository}
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
