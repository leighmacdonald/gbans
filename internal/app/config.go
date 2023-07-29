// Package config contains the functionality for reading in and loosely validating config files.
// The configuration is exposed via package level public variables. These must never be changed
// on the fly and instead configured via the config file or env vars
//
// Env variables will override the config values. They can all be set using the same format as shown to
// map to the correct config keys:
//
//	export discord.token=TOKEN_TOKEN_TOKEN_TOKEN_TOKEN
//	export general.steam_key=STEAM_KEY_STEAM_KEY_STEAM_KEY
//	./gbans serve
package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type filterConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	PingDiscord bool `mapstructure:"ping_discord"`
}

type Config struct {
	General generalConfig `mapstructure:"general"`
	HTTP    httpConfig    `mapstructure:"http"`
	Filter  filterConfig  `mapstructure:"word_filter"`
	DB      dbConfig      `mapstructure:"database"`
	Discord discordConfig `mapstructure:"discord"`
	Log     LogConfig     `mapstructure:"logging"`
	NetBans netBans       `mapstructure:"network_bans"`
	Debug   debugConfig   `mapstructure:"debug"`
	Patreon patreonConfig `mapstructure:"patreon"`
}

type dbConfig struct {
	DSN          string        `mapstructure:"dsn"`
	AutoMigrate  bool          `mapstructure:"auto_migrate"`
	LogQueries   bool          `mapstructure:"log_queries"`
	LogWriteFreq time.Duration `mapstructure:"log_write_freq"`
}

type patreonConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	ClientID            string `mapstructure:"client_id"`
	ClientSecret        string `mapstructure:"client_secret"`
	CreatorAccessToken  string `mapstructure:"creator_access_token"`
	CreatorRefreshToken string `mapstructure:"creator_refresh_token"`
}

type httpConfig struct {
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	TLS                   bool   `mapstructure:"tls"`
	TLSAuto               bool   `mapstructure:"tls_auto"`
	StaticPath            string `mapstructure:"static_path"`
	CookieKey             string `mapstructure:"cookie_key"`
	ClientTimeout         string `mapstructure:"client_timeout"`
	ClientTimeoutDuration time.Duration
	CorsOrigins           []string `mapstructure:"cors_origins"`
}

// Addr returns the address in host:port format.
func (h httpConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type RunMode string

const (
	// ReleaseMode is production mode, minimal logging.
	ReleaseMode RunMode = "release"
	// DebugMode has much more logging and uses non-embedded assets.
	DebugMode RunMode = "debug"
	// TestMode is for unit tests.
	TestMode RunMode = "test"
)

// String returns the string value of the RunMode.
func (rm RunMode) String() string {
	return string(rm)
}

type Action string

const (
	Gag  Action = "gag"
	Kick Action = "kick"
	Ban  Action = "ban"
)

type generalConfig struct {
	SiteName                     string        `mapstructure:"site_name"`
	SteamKey                     string        `mapstructure:"steam_key"`
	Owner                        steamid.SID64 `mapstructure:"owner"`
	Mode                         RunMode       `mapstructure:"mode"`
	WarningTimeout               time.Duration `mapstructure:"warning_timeout"`
	WarningLimit                 int           `mapstructure:"warning_limit"`
	WarningExceededAction        Action        `mapstructure:"warning_exceeded_action"`
	WarningExceededDurationValue string        `mapstructure:"warning_exceeded_duration"`
	WarningExceededDuration      time.Duration `mapstructure:"-"`
	UseUTC                       bool          `mapstructure:"use_utc"`
	ServerStatusUpdateFreq       string        `mapstructure:"server_status_update_freq"`
	MasterServerStatusUpdateFreq string        `mapstructure:"master_server_status_update_freq"`
	DefaultMaps                  []string      `mapstructure:"default_maps"`
	ExternalURL                  string        `mapstructure:"external_url"`
	BannedSteamGroupIds          []steamid.GID `mapstructure:"banned_steam_group_ids"`
	BannedServersAddresses       []string      `mapstructure:"banned_server_addresses"`
}

type discordConfig struct {
	Enabled                bool   `mapstructure:"enabled"`
	AppID                  string `mapstructure:"app_id"`
	AppSecret              string `mapstructure:"app_secret"`
	LinkID                 string `mapstructure:"link_id"`
	Token                  string `mapstructure:"token"`
	GuildID                string `mapstructure:"guild_id"`
	LogChannelID           string `mapstructure:"log_channel_id"`
	PublicLogChannelEnable bool   `mapstructure:"public_log_channel_enable"`
	PublicLogChannelID     string `mapstructure:"public_log_channel_id"`
	ModPingRoleID          string `mapstructure:"mod_ping_role_id"`
	UnregisterOnStart      bool   `mapstructure:"unregister_on_start"`
}

type LogConfig struct {
	Level                string `mapstructure:"level"`
	File                 string `mapstructure:"file"`
	ReportCaller         bool   `mapstructure:"report_caller"`
	FullTimestamp        bool   `mapstructure:"full_timestamp"`
	SrcdsLogAddr         string `mapstructure:"srcds_log_addr"`
	SrcdsLogExternalHost string `mapstructure:"srcds_log_external_host"`
}

type debugConfig struct {
	UpdateSRCDSLogSecrets   bool   `mapstructure:"update_srcds_log_secrets"`
	SkipOpenIDValidation    bool   `mapstructure:"skip_open_id_validation"`
	WriteUnhandledLogEvents bool   `mapstructure:"write_unhandled_log_events"`
	AddRCONLogAddress       string `mapstructure:"add_rcon_log_address"`
}

type netBans struct {
	Enabled     bool                 `mapstructure:"enabled"`
	MaxAge      string               `mapstructure:"max_age"`
	CachePath   string               `mapstructure:"cache_path"`
	Sources     []thirdparty.BanList `mapstructure:"sources"`
	IP2Location ip2locationConf      `mapstructure:"ip2location"`
}

type ip2locationConf struct {
	Enabled      bool   `mapstructure:"enabled"`
	Token        string `mapstructure:"token"`
	ASNEnabled   bool   `mapstructure:"asn_enabled"`
	IPEnabled    bool   `mapstructure:"ip_enabled"`
	ProxyEnabled bool   `mapstructure:"proxy_enabled"`
}

// Read reads in config file and ENV variables if set.
func Read(conf *Config) error {
	const (
		defaultWarnDuration = time.Hour * 24 * 7
		defaultHTTPTimeout  = time.Second * 10
	)

	setDefaultConfigValues()

	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return errors.Wrapf(errReadConfig, "Failed to read config file")
	}

	if errUnmarshal := viper.Unmarshal(conf); errUnmarshal != nil {
		return errors.Wrap(errUnmarshal, "Invalid config file format")
	}

	if strings.HasPrefix(conf.DB.DSN, "pgx://") {
		conf.DB.DSN = strings.Replace(conf.DB.DSN, "pgx://", "postgres://", 1)
	}

	clientDuration, errClientDuration := store.ParseDuration(conf.HTTP.ClientTimeout)
	if errClientDuration != nil {
		clientDuration = defaultHTTPTimeout
	}

	conf.HTTP.ClientTimeoutDuration = clientDuration

	warningDuration, errWarningDuration := store.ParseDuration(conf.General.WarningExceededDurationValue)
	if errWarningDuration != nil {
		warningDuration = defaultWarnDuration
	}

	conf.General.WarningExceededDuration = warningDuration

	gin.SetMode(conf.General.Mode.String())

	if errSteam := steamid.SetKey(conf.General.SteamKey); errSteam != nil {
		return errors.Wrap(errSteam, "Failed to set steamid api key")
	}

	if errSteamWeb := steamweb.SetKey(conf.General.SteamKey); errSteamWeb != nil {
		return errors.Wrap(errSteamWeb, "Failed to set steamweb api key")
	}

	_, errDuration := time.ParseDuration(conf.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		return errors.Errorf("Failed to parse server_status_update_freq: %v", errDuration)
	}

	_, errMaterDuration := time.ParseDuration(conf.General.MasterServerStatusUpdateFreq)
	if errMaterDuration != nil {
		return errors.Wrap(errMaterDuration, "Failed to parse mater_server_status_update_freq")
	}

	return nil
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
		"general.warning_timeout":                  time.Hour * 24,
		"general.warning_limit":                    1,
		"general.warning_exceeded_action":          Gag,
		"general.warning_exceeded_duration":        "168h",
		"general.use_utc":                          true,
		"general.server_status_update_freq":        "60s",
		"general.master_server_status_update_freq": "1m",
		"general.default_maps":                     []string{"pl_badwater"},
		"general.external_url":                     "http://gbans.localhost:6006",
		"general.banned_steam_group_ids":           []steamid.GID{},
		"general.banned_server_addresses":          []string{},
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
		"http.cookie_key":                          golib.RandomString(32),
		"http.client_timeout":                      "10s",
		"debug.update_srcds_log_secrets":           true,
		"debug.skip_open_id_validation":            false,
		"debug.write_unhandled_log_events":         false,
		"filter.enabled":                           false,
		"filter.ping_discord":                      false,
		"discord.enabled":                          false,
		"discord.app_id":                           0,
		"discord.app_secret":                       "",
		"discord.token":                            "",
		"discord.link_id":                          "",
		"discord.perms":                            125958,
		"discord.guild_id":                         "",
		"discord.public_log_channel_enable":        false,
		"discord.public_log_channel_id":            "",
		"discord.log_channel_id":                   "",
		"discord.mod_ping_role_id":                 "",
		"discord.unregister_on_start":              false,
		"network_bans.enabled":                     false,
		"network_bans.max_age":                     "1d",
		"network_bans.cache_path":                  ".cache",
		"network_bans.sources":                     nil,
		"network_bans.ip2location.enabled":         false,
		"network_bans.ip2location.token":           "",
		"network_bans.ip2location.asn_enabled":     false,
		"network_bans.ip2location.ip_enabled":      false,
		"network_bans.ip2location.proxy_enabled":   false,
		"log.level":                                "info",
		"log.report_caller":                        false,
		"log.full_timestamp":                       false,
		"log.srcds_log_addr":                       ":27115",
		"log.srcds_log_external_host":              "",
		"database.dsn":                             "postgresql://gbans:gbans@localhost/gbans",
		"database.auto_migrate":                    true,
		"database.log_queries":                     false,
		"database.log_write_freq":                  time.Second * 10,
	}

	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}
}
