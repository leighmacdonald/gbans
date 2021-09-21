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
	LogAddr string `mapstructure:"log_addr"`
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
}

// Addr returns the address in host:port format
func (h httpConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type runMode string

const (
	// Release is production mode, minimal logging
	Release runMode = "release"
	// Debug has much more logging and uses non-embedded assets
	Debug runMode = "debug"
	// Test is for unit tests
	Test runMode = "test"
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
	DefaultMap                   string        `mapstructure:"default_map"`
	MapChangerEnabled            bool          `mapstructure:"map_changer_enabled"`
}

type discordConfig struct {
	Enabled                bool     `mapstructure:"enabled"`
	AppID                  string   `mapstructure:"app_id"`
	Token                  string   `mapstructure:"token"`
	ModRoleID              string   `mapstructure:"mod_role_id"`
	GuildID                string   `mapstructure:"guild_id"`
	Perms                  int      `mapstructure:"perms"`
	Prefix                 string   `mapstructure:"prefix"`
	ModChannels            []string `mapstructure:"mod_channel_ids"`
	LogChannelID           string   `mapstructure:"log_channel_id"`
	PublicLogChannelEnable bool     `mapstructure:"public_log_channel_enable"`
	PublicLogChannelId     string   `mapstructure:"public_log_channel_id"`
}

type logConfig struct {
	Level          string `mapstructure:"level"`
	ForceColours   bool   `mapstructure:"force_colours"`
	DisableColours bool   `mapstructure:"disable_colours"`
	ReportCaller   bool   `mapstructure:"report_caller"`
	FullTimestamp  bool   `mapstructure:"full_timestamp"`
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
	RPC     rpcConfig
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
	RPC = cfg.RPC

	configureLogger(log.StandardLogger())
	gin.SetMode(General.Mode.String())
	if errSteam := steamid.SetKey(General.SteamKey); errSteam != nil {
		log.Errorf("Failed to set steam api key: %v", err)
	}
	if errSteamWeb := steamweb.SetKey(General.SteamKey); errSteamWeb != nil {
		log.Errorf("Failed to set steam api key: %v", err)
	}
	if found {
		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Warnf("No configuration found, defaults used")
	}
}

func init() {
	viper.SetDefault("general.site_name", "gbans")
	viper.SetDefault("general.steam_key", "")
	viper.SetDefault("general.mode", "release")
	viper.SetDefault("general.owner", 76561198044052046)
	viper.SetDefault("general.warning_timeout", time.Hour*6)
	viper.SetDefault("general.warning_limit", 3)
	viper.SetDefault("general.warning_exceeded_action", Kick)
	viper.SetDefault("general.warning_exceeded_duration", "1w")
	viper.SetDefault("general.use_utc", true)
	viper.SetDefault("general.server_status_update_freq", "60s")
	viper.SetDefault("general.default_map", "pl_badwater")
	viper.SetDefault("general.map_changer_enabled", false)

	viper.SetDefault("http.host", "127.0.0.1")
	viper.SetDefault("http.port", 6006)
	viper.SetDefault("http.domain", "http://localhost:6006")
	viper.SetDefault("http.tls", false)
	viper.SetDefault("http.tls_auto", false)
	viper.SetDefault("http.static_path", "frontend/dist")
	viper.SetDefault("http.cookie_key", golib.RandomString(32))
	viper.SetDefault("http.client_timeout", "10s")

	viper.SetDefault("rpc.addr", "0.0.0.0:7779")
	viper.SetDefault("rpc.enabled", true)

	viper.SetDefault("filter.enabled", false)
	viper.SetDefault("filter.is_warning", true)
	viper.SetDefault("filter.ping_discord", false)
	viper.SetDefault("filter.external_enabled", false)
	viper.SetDefault("filter.external_source", []string{})

	viper.SetDefault("discord.enabled", false)
	viper.SetDefault("discord.app_id", 0)
	viper.SetDefault("discord.token", "")
	viper.SetDefault("discord.mod_role_id", 0)
	viper.SetDefault("discord.perms", 125958)
	viper.SetDefault("discord.prefix", "!")
	viper.SetDefault("discord.mod_channel_ids", nil)
	viper.SetDefault("discord.guild_id", "")
	viper.SetDefault("discord.public_log_channel_enable", false)
	viper.SetDefault("discord.public_log_channel_id", "")

	viper.SetDefault("network_bans.enabled", false)
	viper.SetDefault("network_bans.max_age", "1d")
	viper.SetDefault("network_bans.cache_path", ".cache")
	viper.SetDefault("network_bans.sources", nil)

	viper.SetDefault("network_bans.ip2location.enabled", false)
	viper.SetDefault("network_bans.ip2location.token", "")
	viper.SetDefault("network_bans.ip2location.asn_enabled", false)
	viper.SetDefault("network_bans.ip2location.ip_enabled", false)
	viper.SetDefault("network_bans.ip2location.proxy_enabled", false)

	viper.SetDefault("relay.enabled", false)
	viper.SetDefault("relay.host", "wss://localhost:6006")
	viper.SetDefault("relay.password", "")
	viper.SetDefault("relay.server_name", "")
	viper.SetDefault("relay.log_path", "serverfiles/tf/logs")
	viper.SetDefault("relay.channel_ids", []string{})

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.force_colours", true)
	viper.SetDefault("log.disable_colours", false)
	viper.SetDefault("log.report_caller", false)
	viper.SetDefault("log.full_timestamp", false)

	viper.SetDefault("database.dsn", "postgresql://localhost/gbans")
	viper.SetDefault("database.auto_migrate", true)
	viper.SetDefault("database.log_queries", false)
	viper.SetDefault("database.log_write_freq", time.Second*10)
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
