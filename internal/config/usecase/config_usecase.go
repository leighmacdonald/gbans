package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"strings"
)

type configUsecase struct {
	configRepo domain.ConfigRepository
}

func NewConfigUsecase(conf domain.ConfigRepository) domain.ConfigUsecase {
	config.SetDefaultConfigValues()

	return &configUsecase{configRepo: conf}
}

func (c *configUsecase) ExtURL(obj domain.LinkablePath) string {
	return c.configRepo.Config().ExtURL(obj)
}

func (c *configUsecase) ExtURLRaw(path string, args ...any) string {
	return c.configRepo.Config().ExtURLRaw(path, args...)
}

func (c *configUsecase) Config() *domain.Config {
	return c.configRepo.Config()
}

// Read reads in config file and ENV variables if set.
func (c *configUsecase) Read(conf *domain.Config, noFileOk bool) error {

	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil && !noFileOk {
		return errors.Join(errReadConfig, domain.ErrReadConfig)
	}

	if errUnmarshal := viper.Unmarshal(conf, viper.DecodeHook(mapstructure.DecodeHookFunc(config.DecodeDuration()))); errUnmarshal != nil {
		return errors.Join(errUnmarshal, domain.ErrFormatConfig)
	}

	if strings.HasPrefix(conf.DB.DSN, "pgx://") {
		conf.DB.DSN = strings.Replace(conf.DB.DSN, "pgx://", "postgres://", 1)
	}

	gin.SetMode(conf.General.Mode.String())

	if errSteam := steamid.SetKey(conf.General.SteamKey); errSteam != nil {
		return errors.Join(errSteam, domain.ErrSteamAPIKey)
	}

	if errSteamWeb := steamweb.SetKey(conf.General.SteamKey); errSteamWeb != nil {
		return errors.Join(errSteamWeb, domain.ErrSteamAPIKey)
	}

	return nil
}
