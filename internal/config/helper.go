package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// decodeDuration automatically parses the string duration type (1s,1m,1h,etc.) into a real time.Duration type.
func decodeDuration() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, target reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		// t.TypeOf doesn't seem to work with time.Duration, so just grug it.
		if !strings.HasSuffix(target.String(), "Duration") && !strings.HasSuffix(target.String(), "Freq") {
			return data, nil
		}

		duration, errDuration := datetime.ParseUserStringDuration(data.(string))
		if errDuration != nil {
			return nil, errors.Join(errDuration, fmt.Errorf("%w: %s", domain.ErrDecodeDuration, target.String()))
		}

		return duration, nil
	}
}

func setDefaultConfigValues() {
	if home, errHomeDir := homedir.Dir(); errHomeDir != nil {
		viper.AddConfigPath(home)
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("gbans")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("gbans")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	defaultConfig := map[string]any{
		"owner":                 "",
		"external_url":          "",
		"steam_key":             "",
		"http_host":             "127.0.0.1",
		"http_port":             6006,
		"http_static_path":      "frontend/dist",
		"http_cookie_key":       stringutil.SecureRandomString(32),
		"http_client_timeout":   "10",
		"http_cors_enabled":     true,
		"http_cors_origins":     []string{"http://gbans.localhost"},
		"database_dsn":          "postgresql://gbans:gbans@localhost/gbans",
		"database_auto_migrate": true,
		"database_log_queries":  false,
		"prometheus_enabled":    false,
		"pprof_enabled":         false,
	}

	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}

	if errWriteConfig := viper.SafeWriteConfig(); errWriteConfig != nil {
		return
	}
}
