// Package config contains the functionality for reading in and loosely validating config files.
// The configuration is exposed via package level public variables. These must never be changed
// on the fly and instead configured via the config file or env vars
//
// Env variables will override the config values. They can all be set using the same format as shown to
// map to the correct config keys:
//
// 		export discord.token=TOKEN_TOKEN_TOKEN_TOKEN_TOKEN
// 		export general.steam_key=STEAM_KEY_STEAM_KEY_STEAM_KEY
// 		./gbans serve
//
package config

import (
	"fmt"
	"github.com/leighmacdonald/steamweb"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// BanListType is the type or source of a ban list
type BanListType string

const (
	// CIDR formatted list
	CIDR BanListType = "cidr"
	// ValveNet is the srcds network ban list format
	ValveNet BanListType = "valve_net"
	// ValveSID is the srcds steamid ban list format
	ValveSID BanListType = "valve_steamid"
	// TF2BD sources ban list
	TF2BD BanListType = "tf2bd"
)

// BanList holds details to load a ban lost
type BanList struct {
	URL  string      `mapstructure:"url"`
	Name string      `mapstructure:"name"`
	Type BanListType `mapstructure:"type"`
}

type relayConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Host       string   `mapstructure:"host"`
	Password   string   `mapstructure:"password"`
	ServerName string   `mapstructure:"server_name"`
	LogPath    string   `mapstructure:"log_path"`
	ChannelIDs []string `mapstructure:"channel_ids"`
}

type rpcConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Addr    string `mapstructure:"addr"`
}

type filterConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	IsWarning       bool     `mapstructure:"is_warning"`
	PingDiscord     bool     `mapstructure:"ping_discord"`
	ExternalEnabled bool     `mapstructure:"external_enabled"`
	ExternalSource  []string `mapstructure:"external_source"`
}

type rootConfig struct {
	General generalConfig `mapstructure:"general"`
	RPC     rpcConfig     `mapstructure:"rpc"`
	HTTP    httpConfig    `mapstructure:"http"`
	Relay   relayConfig   `mapstructure:"relay"`
	Filter  filterConfig  `mapstructure:"word_filter"`
	DB      dbConfig      `mapstructure:"database"`
	Discord discordConfig `mapstructure:"discord"`
	Log     logConfig     `mapstructure:"logging"`
	NetBans netBans       `mapstructure:"network_bans"`
	Debug   debugConfig   `mapstructure:"debug"`
}

type dbConfig struct {
	DSN          string        `mapstructure:"dsn"`
	AutoMigrate  bool          `mapstructure:"auto_migrate"`
	LogQueries   bool          `mapstructure:"log_queries"`
	LogWriteFreq time.Duration `mapstructure:"log_write_freq"`
}

type httpConfig struct {
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	Domain                string `mapstructure:"domain"`
	TLS                   bool   `mapstructure:"tls"`
	TLSAuto               bool   `mapstructure:"tls_auto"`
	StaticPath            string `mapstructure:"static_path"`
	CookieKey             string `mapstructure:"cookie_key"`
	ClientTimeout         string `mapstructure:"client_timeout"`
	ClientTimeoutDuration time.Duration
	CorsOrigins           []string `mapstructure:"cors_origins"`
}

// Addr returns the address in host:port format
func (h httpConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type runMode string

const (
	// ReleaseMode is production mode, minimal logging
	ReleaseMode runMode = "release"
	// DebugMode has much more logging and uses non-embedded assets
	DebugMode runMode = "debug"
	// TestMode is for unit tests
	TestMode runMode = "test"
)

// String returns the string value of the runMode
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
	DefaultMaps                  []string      `mapstructure:"default_maps"`
	MapChangerEnabled            bool          `mapstructure:"map_changer_enabled"`
}

type discordConfig struct {
	Enabled                bool     `mapstructure:"enabled"`
	AppID                  string   `mapstructure:"app_id"`
	Token                  string   `mapstructure:"token"`
	ModRoleIDs             []string `mapstructure:"mod_role_ids"`
	GuildID                string   `mapstructure:"guild_id"`
	Perms                  int      `mapstructure:"perms"`
	Prefix                 string   `mapstructure:"prefix"`
	ModChannels            []string `mapstructure:"mod_channel_ids"`
	LogChannelID           string   `mapstructure:"log_channel_id"`
	PublicLogChannelEnable bool     `mapstructure:"public_log_channel_enable"`
	PublicLogChannelId     string   `mapstructure:"public_log_channel_id"`
}

type logConfig struct {
	Level                string `mapstructure:"level"`
	ForceColours         bool   `mapstructure:"force_colours"`
	DisableColours       bool   `mapstructure:"disable_colours"`
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

// Default config values. Anything defined in the config or env will override them
var (
	General generalConfig
	HTTP    httpConfig
	Filter  filterConfig
	Relay   relayConfig
	DB      dbConfig
	Discord discordConfig
	Log     logConfig
	Net     netBans
	Debug   debugConfig
)

// Read reads in config file and ENV variables if set.
func Read(cfgFiles ...string) {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Failed to get HOME dir: %v", err)
	}
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName("gbans")
	viper.SetEnvPrefix("gbans")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	found := false
	for _, cfgFile := range cfgFiles {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Failed to read config file: %s", cfgFile)
		}
		found = true
	}
	var cfg rootConfig
	if err2 := viper.Unmarshal(&cfg); err2 != nil {
		log.Fatalf("Invalid config file format: %v", err2)
	}
	if strings.HasPrefix(cfg.DB.DSN, "pgx://") {
		cfg.DB.DSN = strings.Replace(cfg.DB.DSN, "pgx://", "postgres://", 1)
	}
	d, err3 := ParseDuration(cfg.HTTP.ClientTimeout)
	if err3 != nil {
		d = time.Second * 10
	}
	cfg.HTTP.ClientTimeoutDuration = d
	d2, err4 := ParseDuration(cfg.General.WarningExceededDurationValue)
	if err4 != nil {
		d2 = time.Hour * 24 * 7
	}
	cfg.General.WarningExceededDuration = d2
	HTTP = cfg.HTTP
	General = cfg.General
	Filter = cfg.Filter
	Discord = cfg.Discord
	Relay = cfg.Relay
	DB = cfg.DB
	Log = cfg.Log
	Net = cfg.NetBans
	Debug = cfg.Debug

	configureLogger(log.StandardLogger())
	gin.SetMode(General.Mode.String())
	if errSteam := steamid.SetKey(General.SteamKey); errSteam != nil {
		log.Errorf("Failed to set steam api key: %v", err)
	}
	if errSteamWeb := steamweb.SetKey(General.SteamKey); errSteamWeb != nil {
		log.Errorf("Failed to set steam api key: %v", err)
	}
	if found {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Warnf("No configuration found, defaults used")
	}
}

var defaultConfig = map[string]any{
	"general.site_name":                      "gbans",
	"general.steam_key":                      "",
	"general.mode":                           "release",
	"general.owner":                          76561198044052046,
	"general.warning_timeout":                time.Hour * 6,
	"general.warning_limit":                  3,
	"general.warning_exceeded_action":        Kick,
	"general.warning_exceeded_duration":      "1w",
	"general.use_utc":                        true,
	"general.server_status_update_freq":      "60s",
	"general.default_maps":                   []string{"pl_badwater"},
	"general.map_changer_enabled":            false,
	"http.host":                              "127.0.0.1",
	"http.port":                              6006,
	"http.domain":                            "http://localhost:6006",
	"http.tls":                               false,
	"http.tls_auto":                          false,
	"http.static_path":                       "frontend/dist",
	"http.cookie_key":                        golib.RandomString(32),
	"http.client_timeout":                    "10s",
	"debug.update_srcds_log_secrets":         true,
	"debug.skip_open_id_validation":          false,
	"debug.write_unhandled_log_events":       false,
	"filter.enabled":                         false,
	"filter.is_warning":                      true,
	"filter.ping_discord":                    false,
	"filter.external_enabled":                false,
	"filter.external_source":                 []string{},
	"discord.enabled":                        false,
	"discord.app_id":                         0,
	"discord.token":                          "",
	"discord.mod_role_ids":                   []string{},
	"discord.perms":                          125958,
	"discord.mod_channel_ids":                nil,
	"discord.guild_id":                       "",
	"discord.public_log_channel_enable":      false,
	"discord.public_log_channel_id":          "",
	"network_bans.enabled":                   false,
	"network_bans.max_age":                   "1d",
	"network_bans.cache_path":                ".cache",
	"network_bans.sources":                   nil,
	"network_bans.ip2location.enabled":       false,
	"network_bans.ip2location.token":         "",
	"network_bans.ip2location.asn_enabled":   false,
	"network_bans.ip2location.ip_enabled":    false,
	"network_bans.ip2location.proxy_enabled": false,
	"log.level":                              "info",
	"log.force_colours":                      true,
	"log.disable_colours":                    false,
	"log.report_caller":                      false,
	"log.full_timestamp":                     false,
	"log.srcds_log_addr":                     ":27115",
	"log.srcds_log_external_host":            "",
	"database.dsn":                           "postgresql://localhost/gbans",
	"database.auto_migrate":                  true,
	"database.log_queries":                   false,
	"database.log_write_freq":                time.Second * 10,
}

func init() {
	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}
}

func configureLogger(l *log.Logger) {
	level, err := log.ParseLevel(Log.Level)
	if err != nil {
		log.Debugf("Invalid log level: %s", Log.Level)
		level = log.InfoLevel
	}
	l.SetLevel(level)
	l.SetFormatter(&log.TextFormatter{
		ForceColors:   Log.ForceColours,
		DisableColors: Log.DisableColours,
		FullTimestamp: Log.FullTimestamp,
	})
	l.SetReportCaller(Log.ReportCaller)
}
