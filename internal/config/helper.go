package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
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

		duration, errDuration := util.ParseUserStringDuration(data.(string))
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
		"general.site_name":                        "gbans",
		"general.steam_key":                        "",
		"general.owner":                            "76561198044052046",
		"general.mode":                             domain.ReleaseMode,
		"general.warning_timeout":                  "72h",
		"general.warning_limit":                    2,
		"general.use_utc":                          true,
		"general.server_status_update_freq":        "60s",
		"general.master_server_status_update_freq": "60s",
		"general.default_maps":                     []string{},
		"general.external_url":                     "http://gbans.localhost",
		"general.demo_cleanup_enabled":             true,
		"general.demo_count_limit":                 10000,
		"general.file_serve_mode":                  domain.LocalMode,
		"http.host":                                "127.0.0.1",
		"http.port":                                6006,
		"http.tls":                                 false,
		"http.tls_auto":                            false,
		"http.static_path":                         "frontend/dist",
		"http.cookie_key":                          util.SecureRandomString(32),
		"http.client_timeout":                      "10s",
		"http.cors_origins":                        []string{"http://gbans.localhost"},
		"word_filter.enabled":                      false,
		"word_filter.dry":                          false,
		"word_filter.ping_discord":                 false,
		"word_filter.max_weight":                   6,
		"word_filter.check_timeout":                "1s",
		"word_filter.match_timeout":                "120m",
		"database.dsn":                             "postgresql://gbans:gbans@localhost/gbans",
		"database.auto_migrate":                    true,
		"database.log_queries":                     false,
		"discord.enabled":                          false,
		"discord.app_id":                           "",
		"discord.app_secret":                       "",
		"discord.link_id":                          "",
		"discord.token":                            "",
		"discord.guild_id":                         "",
		"discord.log_channel_id":                   "",
		"discord.public_log_channel_enable":        false,
		"discord.public_log_channel_id":            "",
		"discord.public_match_log_channel_id":      "",
		"discord.mod_ping_role_id":                 "",
		"discord.unregister_on_start":              false,
		"logging.level":                            "info",
		"logging.file":                             "",
		"logging.report_caller":                    false,
		"logging.full_timestamp":                   false,
		"logging.srcds_log_addr":                   ":27115",
		"logging.sentry_dsn":                       "",
		"logging.sentry_dsn_web":                   "",
		"logging.sentry_trace":                     true,
		"logging.sentry_sample_rate":               1.0,
		"ip2location.enabled":                      false,
		"ip2location.cache_path":                   ".cache",
		"ip2location.token":                        "",
		"ip2location.asn_enabled":                  false,
		"ip2location.ip_enabled":                   false,
		"ip2location.proxy_enabled":                false,
		"debug.update_srcds_log_secrets":           true,
		"debug.skip_open_id_validation":            false,
		"debug.write_unhandled_log_events":         false,
		"debug.add_rcon_log_address":               "",
		"patreon.enabled":                          false,
		"patreon.client_id":                        "",
		"patreon.client_secret":                    "",
		"patreon.creator_access_token":             "",
		"patreon.creator_refresh_token":            "",
		"s3.enabled":                               false,
		"s3.access_key":                            "",
		"s3.secret_key":                            "",
		"s3.endpoint":                              "localhost:9001",
		"s3.external_url":                          "http://minio.localhost",
		"s3.ssl":                                   false,
		"s3.region":                                "",
		"s3.bucket_media":                          "media",
		"s3.bucket_demo":                           "demos",
		"ssh.enabled":                              false,
		"ssh.username":                             "",
		"ssh.port":                                 22,
		"ssh.private_key_path":                     "",
		"ssh.password":                             "",
		"ssh.update_interval":                      "60s",
		"ssh.timeout":                              "10s",
		"ssh.demo_path_fmt":                        "",
		"local_store.path_root":                    "./assets",
		"exports.bd_enabled":                       false,
		"exports.valve_enabled":                    false,
		"exports.authorized_keys":                  []string{},
	}

	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}

	if errWriteConfig := viper.SafeWriteConfig(); errWriteConfig != nil {
		return
	}
}
