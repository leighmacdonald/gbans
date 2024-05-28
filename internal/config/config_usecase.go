package config

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

type configUsecase struct {
	configRepo    domain.ConfigRepository
	static        domain.StaticConfig
	configMu      sync.RWMutex
	currentConfig domain.Config
}

func NewConfigUsecase(static domain.StaticConfig, repository domain.ConfigRepository) domain.ConfigUsecase {
	return &configUsecase{static: static, configRepo: repository}
}

func (c *configUsecase) Init(ctx context.Context) error {
	return c.configRepo.Init(ctx)
}

func (c *configUsecase) Write(ctx context.Context, config domain.Config) error {
	return c.configRepo.Write(ctx, config)
}

func (c *configUsecase) ExtURL(obj domain.LinkablePath) string {
	return c.Config().ExtURL(obj)
}

func (c *configUsecase) ExtURLRaw(path string, args ...any) string {
	return c.Config().ExtURLRaw(path, args...)
}

func (c *configUsecase) Config() domain.Config {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	return c.currentConfig
}

func (c *configUsecase) Reload(ctx context.Context) error {
	config, errConfig := c.configRepo.Read(ctx)
	if errConfig != nil {
		return errConfig
	}

	if err := applyGlobalConfig(config); err != nil {
		return err
	}

	config.StaticConfig = c.static

	c.configMu.Lock()
	c.currentConfig = config
	c.configMu.Unlock()

	return nil
}

func ReadStaticConfig() (domain.StaticConfig, error) {
	setDefaultConfigValues()

	var config domain.StaticConfig
	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return config, errors.Join(errReadConfig, domain.ErrReadConfig)
	}

	if errUnmarshal := viper.Unmarshal(&config, viper.DecodeHook(mapstructure.DecodeHookFunc(decodeDuration()))); errUnmarshal != nil {
		return config, errors.Join(errUnmarshal, domain.ErrFormatConfig)
	}

	if strings.HasPrefix(config.DatabaseDSN, "pgx://") {
		config.DatabaseDSN = strings.Replace(config.DatabaseDSN, "pgx://", "postgres://", 1)
	}

	ownerSID := steamid.New(config.Owner)
	if !ownerSID.Valid() {
		return config, domain.ErrInvalidSID
	}

	return config, nil
}

func applyGlobalConfig(config domain.Config) error {
	gin.SetMode(config.General.Mode.String())

	if errSteam := steamid.SetKey(config.General.SteamKey); errSteam != nil {
		return errors.Join(errSteam, domain.ErrSteamAPIKey)
	}

	if errSteamWeb := steamweb.SetKey(config.General.SteamKey); errSteamWeb != nil {
		return errors.Join(errSteamWeb, domain.ErrSteamAPIKey)
	}

	return nil
}
