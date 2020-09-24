package config

import "C"
import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type rootConfig struct {
	HTTP    HTTPConfig    `mapstructure:"http"`
	DB      DBConfig      `mapstructure:"database"`
	Discord DiscordConfig `mapstructure:"discord"`
	Log     LogConfig     `mapstructure:"logging"`
}

type DBConfig struct {
	Path string `mapstructure:"path"`
}

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type DiscordConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
	Perms   int    `mapstructure:"perms"`
}

type LogConfig struct {
	Level          string `mapstructure:"level"`
	ForceColours   bool   `mapstructure:"force_colours"`
	DisableColours bool   `mapstructure:"disable_colours"`
	ReportCaller   bool   `mapstructure:"report_caller"`
	FullTimestamp  bool   `mapstructure:"full_timestamp"`
}

// Default config values. Anything defined in the config or env will overwrite them
var (
	HTTP = HTTPConfig{
		Host: "127.0.0.1",
		Port: 6970,
		Mode: "release",
	}
	DB = DBConfig{
		Path: "db.sqlite",
	}
	Discord = DiscordConfig{
		Enabled: false,
		Token:   "",
		// Kick / Ban / Send Msg / Manage msg / embed / attach file / read history
		Perms: 125958,
	}
	Log = LogConfig{
		Level:          "info",
		DisableColours: false,
		ForceColours:   false,
		ReportCaller:   false,
		FullTimestamp:  false,
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
			fmt.Println(err)
			os.Exit(1)
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
		HTTP = cfg.HTTP
		Discord = cfg.Discord
		DB = cfg.DB
		Log = cfg.Log
		found = true
	}
	configureLogger(log.StandardLogger())
	gin.SetMode(HTTP.Mode)
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
