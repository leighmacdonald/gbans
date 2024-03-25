package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
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
		"general.mode":                             "release",
		"general.owner":                            76561198044052046,
		"general.warning_timeout":                  "72h",
		"general.warning_limit":                    2,
		"general.warning_exceeded_action":          domain.ActionGag,
		"general.warning_exceeded_duration":        "168h",
		"general.use_utc":                          true,
		"general.server_status_update_freq":        "60s",
		"general.master_server_status_update_freq": "1m",
		"general.external_url":                     "http://gbans.localhost:6006",
		"general.banned_steam_group_ids":           []steamid.SteamID{},
		"general.banned_server_addresses":          []string{},
		"general.demo_cleanup_enabled":             true,
		"general.demo_count_limit":                 10000,
		"patreon.enabled":                          false,
		"patreon.client_id":                        "",
		"patreon.client_secret":                    "",
		"patreon.creator_access_token":             "",
		"patreon.creator_refresh_token":            "",
		"http.host":                                "127.0.0.1",
		"http.port":                                6006,
		"http.tls":                                 false,
		"http.tls_auto":                            false,
		"http.static_path":                         "frontend/dist",
		"http.cookie_key":                          util.SecureRandomString(32),
		"http.client_timeout":                      "10s",
		"debug.update_srcds_log_secrets":           true,
		"debug.skip_open_id_validation":            false,
		"debug.write_unhandled_log_events":         false,
		"word_filter.enabled":                      false,
		"word_filter.dry":                          false,
		"word_filter.ping_discord":                 false,
		"word_filter.max_weight":                   6,
		"word_filter.match_timeout":                "120m",
		"word_filter.check_timeout":                "1s",
		"discord.enabled":                          false,
		"discord.app_id":                           0,
		"discord.app_secret":                       "",
		"discord.token":                            "",
		"discord.link_id":                          "",
		"discord.perms":                            125958,
		"discord.guild_id":                         "",
		"discord.public_log_channel_enable":        false,
		"discord.public_log_channel_id":            "",
		"discord.public_match_log_channel_id":      "",
		"discord.log_channel_id":                   "",
		"discord.mod_ping_role_id":                 "",
		"discord.unregister_on_start":              false,
		"ip2location.enabled":                      false,
		"ip2location.token":                        "",
		"ip2location.asn_enabled":                  false,
		"ip2location.ip_enabled":                   false,
		"ip2location.proxy_enabled":                false,
		"log.level":                                "info",
		"log.report_caller":                        false,
		"log.full_timestamp":                       false,
		"log.srcds_log_addr":                       ":27115",
		"log.sentry_dsn":                           "",
		"log.sentry_dsn_web":                       "",
		"log.sentry_trace":                         true,
		"log.sentry_sample_rate":                   1.0,
		"database.dsn":                             "postgresql://gbans:gbans@localhost/gbans",
		"database.auto_migrate":                    true,
		"database.log_queries":                     false,
		"s3.enabled":                               false,
		"s3.access_key":                            "",
		"s3.secret_key":                            "",
		"s3.endpoint":                              "localhost:9001",
		"s3.ssl":                                   false,
		"s3.region":                                "",
		"s3.bucket_media":                          "media",
		"s3.bucket_demo":                           "demos",
	}

	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}
}
