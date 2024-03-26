package config

import (
	"errors"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type configRepository struct {
	Conf domain.Config
	mu   sync.RWMutex
}

func NewConfigRepository() domain.ConfigRepository {
	return &configRepository{Conf: domain.Config{}}
}

func (c *configRepository) Config() domain.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Conf
}

func (c *configRepository) Read(noFileOk bool) error {
	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil && !noFileOk {
		return errors.Join(errReadConfig, domain.ErrReadConfig)
	}

	var newConfig domain.Config

	if errUnmarshal := viper.Unmarshal(&newConfig, viper.DecodeHook(mapstructure.DecodeHookFunc(decodeDuration()))); errUnmarshal != nil {
		return errors.Join(errUnmarshal, domain.ErrFormatConfig)
	}

	if strings.HasPrefix(newConfig.DB.DSN, "pgx://") {
		newConfig.DB.DSN = strings.Replace(newConfig.DB.DSN, "pgx://", "postgres://", 1)
	}

	gin.SetMode(newConfig.General.Mode.String())

	if errSteam := steamid.SetKey(newConfig.General.SteamKey); errSteam != nil {
		return errors.Join(errSteam, domain.ErrSteamAPIKey)
	}

	if errSteamWeb := steamweb.SetKey(newConfig.General.SteamKey); errSteamWeb != nil {
		return errors.Join(errSteamWeb, domain.ErrSteamAPIKey)
	}

	ownerSID := steamid.New(newConfig.General.Owner)
	if !ownerSID.Valid() {
		return domain.ErrInvalidSID
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Conf = newConfig

	return nil
}
