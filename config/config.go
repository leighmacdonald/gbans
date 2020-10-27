package config

import "C"
import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
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

type rootConfig struct {
	General GeneralConfig `mapstructure:"general"`
	HTTP    HTTPConfig    `mapstructure:"http"`
	DB      DBConfig      `mapstructure:"database"`
	Discord DiscordConfig `mapstructure:"discord"`
	Log     LogConfig     `mapstructure:"logging"`
	NetBans NetBans       `mapstructure:"network_bans"`
}

type DBConfig struct {
	Path string `mapstructure:"path"`
}

type HTTPConfig struct {
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	Mode                  string `mapstructure:"mode"`
	Domain                string `mapstructure:"domain"`
	StaticPath            string `mapstructure:"static_path"`
	SiteName              string `mapstructure:"site_name"`
	CookieKey             string `mapstructure:"cookie_key"`
	ClientTimeout         string `mapstructure:"client_timeout"`
	ClientTimeoutDuration time.Duration
}

func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type GeneralConfig struct {
	SteamKey string `mapstructure:"steam_key"`
}

type DiscordConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Token       string   `mapstructure:"token"`
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
	General = GeneralConfig{
		SteamKey: "",
	}
	HTTP = HTTPConfig{
		Host:                  "127.0.0.1",
		Port:                  6970,
		Mode:                  "release",
		Domain:                "http://localhost:6006",
		StaticPath:            "frontend/dist",
		SiteName:              "gbans",
		CookieKey:             golib.RandomString(32),
		ClientTimeout:         "30s",
		ClientTimeoutDuration: time.Second * 30,
	}
	DB = DBConfig{
		Path: "db.sqlite",
	}
	Discord = DiscordConfig{
		Enabled: false,
		Token:   "",
		// Kick / Ban / Send Msg / Manage msg / embed / attach file / read history
		Perms:  125958,
		Prefix: "!",
	}
	Log = LogConfig{
		Level:          "info",
		DisableColours: false,
		ForceColours:   false,
		ReportCaller:   false,
		FullTimestamp:  false,
	}
	Net = NetBans{
		Enabled:   false,
		MaxAge:    "1w",
		CachePath: ".cache",
		Sources:   []BanList{},
	}
)

// Read reads in config file and ENV variables if set.
func Read(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatalf("Failed to get HOME dir: %v", err)
		}
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName("gbans")
	}

	viper.AutomaticEnv()
	found := false
	if err := viper.ReadInConfig(); err == nil {
		var cfg rootConfig
		if err := viper.Unmarshal(&cfg); err != nil {
			log.Fatalf("Invalid config file format: %v", err)
		}
		d, err := util.ParseDuration(cfg.HTTP.ClientTimeout)
		if err != nil {
			log.Fatalf("Could not parse http client timeout duration: %v", err)
		}
		cfg.HTTP.ClientTimeoutDuration = d
		HTTP = cfg.HTTP
		General = cfg.General
		Discord = cfg.Discord
		DB = cfg.DB
		Log = cfg.Log
		Net = cfg.NetBans
		found = true
	}
	configureLogger(log.StandardLogger())
	gin.SetMode(HTTP.Mode)
	steamid.SetKey(General.SteamKey)
	if found {
		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Warnf("No configuration found, defaults used")
	}
}

func configureLogger(l *log.Logger) {
	level, err := log.ParseLevel(Log.Level)
	if err != nil {
		log.Fatalf("Invalid log level: %s", Log.Level)
	}
	l.SetLevel(level)
	l.SetFormatter(&log.TextFormatter{
		ForceColors:   Log.ForceColours,
		DisableColors: Log.DisableColours,
		FullTimestamp: Log.FullTimestamp,
	})
	l.SetReportCaller(Log.ReportCaller)
}
