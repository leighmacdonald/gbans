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
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// BanListType is the type or source of a ban list.
type BanListType string

const (
	// CIDR formatted list.
	CIDR BanListType = "cidr"
	// ValveNet is the srcds network ban list format.
	ValveNet BanListType = "valve_net"
	// ValveSID is the srcds steamid ban list format.
	ValveSID BanListType = "valve_steamid"
	// TF2BD sources ban list.
	TF2BD BanListType = "tf2bd"
)

// BanList holds details to load a ban lost.
type BanList struct {
	URL  string      `mapstructure:"url"`
	Name string      `mapstructure:"name"`
	Type BanListType `mapstructure:"type"`
}

type filterConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	PingDiscord bool `mapstructure:"ping_discord"`
}

type rootConfig struct {
	General generalConfig `mapstructure:"general"`
	HTTP    httpConfig    `mapstructure:"http"`
	Filter  filterConfig  `mapstructure:"word_filter"`
	DB      dbConfig      `mapstructure:"database"`
	Discord discordConfig `mapstructure:"discord"`
	Log     logConfig     `mapstructure:"logging"`
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
	ClientId            string `mapstructure:"client_id"`
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

type runMode string

const (
	// ReleaseMode is production mode, minimal logging.
	ReleaseMode runMode = "release"
	// DebugMode has much more logging and uses non-embedded assets.
	DebugMode runMode = "debug"
	// TestMode is for unit tests.
	TestMode runMode = "test"
)

// String returns the string value of the runMode.
func (rm runMode) String() string {
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
	Mode                         runMode       `mapstructure:"mode"`
	WarningTimeout               time.Duration `mapstructure:"warning_timeout"`
	WarningLimit                 int           `mapstructure:"warning_limit"`
	WarningExceededAction        Action        `mapstructure:"warning_exceeded_action"`
	WarningExceededDurationValue string        `mapstructure:"warning_exceeded_duration"`
	WarningExceededDuration      time.Duration `mapstructure:"-"`
	UseUTC                       bool          `mapstructure:"use_utc"`
	ServerStatusUpdateFreq       string        `mapstructure:"server_status_update_freq"`
	MasterServerStatusUpdateFreq string        `mapstructure:"master_server_status_update_freq"`
	DefaultMaps                  []string      `mapstructure:"default_maps"`
	DemoRootPath                 string        `mapstructure:"demo_root_path"`
	ExternalUrl                  string        `mapstructure:"external_url"`
	BannedSteamGroupIds          []steamid.GID `mapstructure:"banned_steam_group_ids"`
	BannedServersAddresses       []string      `mapstructure:"banned_server_addresses"`
}

type discordConfig struct {
	Enabled                bool     `mapstructure:"enabled"`
	AppID                  string   `mapstructure:"app_id"`
	AppSecret              string   `mapstructure:"app_secret"`
	LinkId                 string   `mapstructure:"link_id"`
	Token                  string   `mapstructure:"token"`
	ModRoleIDs             []string `mapstructure:"mod_role_ids"`
	GuildID                string   `mapstructure:"guild_id"`
	Perms                  int      `mapstructure:"perms"`
	Prefix                 string   `mapstructure:"prefix"`
	ModChannels            []string `mapstructure:"mod_channel_ids"`
	LogChannelID           string   `mapstructure:"log_channel_id"`
	PublicLogChannelEnable bool     `mapstructure:"public_log_channel_enable"`
	PublicLogChannelId     string   `mapstructure:"public_log_channel_id"`
	ModLogChannelId        string   `mapstructure:"mod_log_channel_id"`
	ReportLogChannelId     string   `mapstructure:"report_log_channel_id"`
}

type logConfig struct {
	Level                string `mapstructure:"level"`
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
	Enabled     bool        `mapstructure:"enabled"`
	MaxAge      string      `mapstructure:"max_age"`
	CachePath   string      `mapstructure:"cache_path"`
	Sources     []BanList   `mapstructure:"sources"`
	IP2Location ip2location `mapstructure:"ip2location"`
}

type ip2location struct {
	Enabled      bool   `mapstructure:"enabled"`
	Token        string `mapstructure:"token"`
	ASNEnabled   bool   `mapstructure:"asn_enabled"`
	IPEnabled    bool   `mapstructure:"ip_enabled"`
	ProxyEnabled bool   `mapstructure:"proxy_enabled"`
}

// Default config values. Anything defined in the config or env will override them.
var (
	General generalConfig
	HTTP    httpConfig
	Filter  filterConfig
	DB      dbConfig
	Discord discordConfig
	Log     logConfig
	Net     netBans
	Debug   debugConfig
	Patreon patreonConfig
)

// Read reads in config file and ENV variables if set.
func Read() (string, error) {
	// Find home directory.
	home, errHomeDir := homedir.Dir()
	if errHomeDir != nil {
		return "", errors.Errorf("Failed to get HOME dir: %v", errHomeDir)
	}
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName("gbans")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("gbans")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return "", errors.Wrapf(errReadConfig, "Failed to read config file")
	}

	var root rootConfig
	if errUnmarshal := viper.Unmarshal(&root); errUnmarshal != nil {
		return "", errors.Wrap(errUnmarshal, "Invalid config file format")
	}
	if strings.HasPrefix(root.DB.DSN, "pgx://") {
		root.DB.DSN = strings.Replace(root.DB.DSN, "pgx://", "postgres://", 1)
	}
	clientDuration, errClientDuration := ParseDuration(root.HTTP.ClientTimeout)
	if errClientDuration != nil {
		clientDuration = time.Second * 10
	}
	root.HTTP.ClientTimeoutDuration = clientDuration
	warningDuration, errWarningDuration := ParseDuration(root.General.WarningExceededDurationValue)
	if errWarningDuration != nil {
		warningDuration = time.Hour * 24 * 7
	}
	if errDemoRoot := os.MkdirAll(root.General.DemoRootPath, 0o775); errDemoRoot != nil {
		return "", errors.Errorf("Failed to create demo_root_path: %v", errDemoRoot)
	}
	root.General.WarningExceededDuration = warningDuration
	HTTP = root.HTTP
	General = root.General
	Filter = root.Filter
	Discord = root.Discord
	DB = root.DB
	Log = root.Log
	Net = root.NetBans
	Debug = root.Debug
	Patreon = root.Patreon

	gin.SetMode(General.Mode.String())
	if errSteam := steamid.SetKey(General.SteamKey); errSteam != nil {
		return "", errors.Errorf("Failed to set steam api key: %v", errHomeDir)
	}
	_, errDuration := time.ParseDuration(General.ServerStatusUpdateFreq)
	if errDuration != nil {
		return "", errors.Errorf("Failed to parse server_status_update_freq: %v", errDuration)
	}
	_, errMaterDuration := time.ParseDuration(General.MasterServerStatusUpdateFreq)
	if errMaterDuration != nil {
		return "", errors.Errorf("Failed to parse mater_server_status_update_freq: %v", errMaterDuration)
	}
	if errSteamWeb := steamweb.SetKey(General.SteamKey); errSteamWeb != nil {
		return "", errors.Errorf("Failed to set steam api key: %v", errHomeDir)
	}
	return viper.ConfigFileUsed(), nil
}

var defaultConfig = map[string]any{
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
	"general.demo_root_path":                   "./.demos/",
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
	"discord.mod_role_ids":                     []string{},
	"discord.perms":                            125958,
	"discord.mod_channel_ids":                  []string{},
	"discord.guild_id":                         "",
	"discord.public_log_channel_enable":        false,
	"discord.public_log_channel_id":            "",
	"discord.report_log_channel_id":            "",
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

func init() {
	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}
}

func ExtURL(path string, args ...any) string {
	return strings.TrimRight(General.ExternalUrl, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}
