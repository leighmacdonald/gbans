package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type BanListType string

const (
	CIDR     BanListType = "cidr"
	Socks5   BanListType = "socks5"
	Socks4   BanListType = "socks4"
	Web      BanListType = "http"
	Snort    BanListType = "snort"
	ValveNet BanListType = "valve_net"
	ValveSID BanListType = "valve_steamid"
	TF2BD    BanListType = "tf2bd"
)

type BanList struct {
	URL  string      `mapstructure:"url"`
	Name string      `mapstructure:"name"`
	Type BanListType `mapstructure:"type"`
}

type RelayConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	ChannelIDs []string `mapstructure:"channel_ids"`
}

type FilterConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	IsWarning       bool     `mapstructure:"is_warning"`
	PingDiscord     bool     `mapstructure:"ping_discord"`
	ExternalEnabled bool     `mapstructure:"external_enabled"`
	ExternalSource  []string `mapstructure:"external_source"`
}

type rootConfig struct {
	General GeneralConfig `mapstructure:"general"`
	HTTP    HTTPConfig    `mapstructure:"http"`
	Relay   RelayConfig   `mapstructure:"relay"`
	Filter  FilterConfig  `mapstructure:"word_filter"`
	DB      DBConfig      `mapstructure:"database"`
	Discord DiscordConfig `mapstructure:"discord"`
	Log     LogConfig     `mapstructure:"logging"`
	NetBans NetBans       `mapstructure:"network_bans"`
}

type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

type HTTPConfig struct {
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

func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type GeneralConfig struct {
	SiteName       string        `mapstructure:"site_name"`
	SteamKey       string        `mapstructure:"steam_key"`
	Owner          steamid.SID64 `mapstructure:"owner"`
	Mode           string        `mapstructure:"mode"`
	WarningTimeout time.Duration `mapstructure:"warning_timeout"`
	WarningLimit   int           `mapstructure:"warning_limit"`
	UseUTC         bool          `mapstructure:"use_utc"`
}

type DiscordConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	AppID       string   `mapstructure:"app_id"`
	Token       string   `mapstructure:"token"`
	ModRoleID   int      `mapstructure:"mod_role_id"`
	Perms       int      `mapstructure:"perms"`
	Prefix      string   `mapstructure:"prefix"`
	ModChannels []string `mapstructure:"mod_channel_ids"`
}

type LogConfig struct {
	Level          string `mapstructure:"level"`
	ForceColours   bool   `mapstructure:"force_colours"`
	DisableColours bool   `mapstructure:"disable_colours"`
	ReportCaller   bool   `mapstructure:"report_caller"`
	FullTimestamp  bool   `mapstructure:"full_timestamp"`
}

type NetBans struct {
	Enabled   bool      `mapstructure:"enabled"`
	MaxAge    string    `mapstructure:"max_age"`
	CachePath string    `mapstructure:"cache_path"`
	Sources   []BanList `mapstructure:"sources"`
}

// Default config values. Anything defined in the config or env will overwrite them
var (
	General GeneralConfig
	HTTP    HTTPConfig
	Filter  FilterConfig
	Relay   RelayConfig
	DB      DBConfig
	Discord DiscordConfig
	Log     LogConfig
	Net     NetBans
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
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Invalid config file format: %v", err)
	}
	d, err := ParseDuration(cfg.HTTP.ClientTimeout)
	if err != nil {
		d = time.Second * 10
	}
	cfg.HTTP.ClientTimeoutDuration = d
	HTTP = cfg.HTTP
	General = cfg.General
	Filter = cfg.Filter
	Discord = cfg.Discord
	Relay = cfg.Relay
	DB = cfg.DB
	Log = cfg.Log
	Net = cfg.NetBans

	configureLogger(log.StandardLogger())
	gin.SetMode(General.Mode)
	steamid.SetKey(General.SteamKey)
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
	viper.SetDefault("general.owner", 76561198084134025)
	viper.SetDefault("general.warning_timeout", time.Hour*6)
	viper.SetDefault("general.warning_limit", 3)
	viper.SetDefault("general.use_utc", true)

	viper.SetDefault("http.host", "127.0.0.1")
	viper.SetDefault("http.port", 6006)
	viper.SetDefault("http.domain", "http://localhost:6006")
	viper.SetDefault("http.tls", false)
	viper.SetDefault("http.tls_auto", false)
	viper.SetDefault("http.static_path", "frontend/dist")
	viper.SetDefault("http.cookie_key", golib.RandomString(32))
	viper.SetDefault("http.client_timeout", "10s")

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

	viper.SetDefault("network_bans.enabled", false)
	viper.SetDefault("network_bans.max_age", "1d")
	viper.SetDefault("network_bans.cache_path", ".cache")
	viper.SetDefault("network_bans.sources", nil)

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.force_colours", true)
	viper.SetDefault("log.disable_colours", false)
	viper.SetDefault("log.report_caller", false)
	viper.SetDefault("log.full_timestamp", false)

	viper.SetDefault("database.dsn", "postgresql://localhost/gbans")
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
