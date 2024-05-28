package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type LinkablePath interface {
	Path() string
}

type ConfigRepository interface {
	Read(ctx context.Context) (Config, error)
	Write(ctx context.Context, config Config) error
	Init(ctx context.Context) error
}

type ConfigUsecase interface {
	Config() Config
	ExtURL(obj LinkablePath) string
	ExtURLRaw(path string, args ...any) string
	Reload(ctx context.Context) error
	Write(ctx context.Context, config Config) error
	Init(ctx context.Context) error
}

// StaticConfig defines non-dynamic config values that cannot be changed during runtime.
type StaticConfig struct {
	Owner               string   `mapstructure:"owner"`
	ExternalURL         string   `mapstructure:"external_url"`
	HTTPHost            string   `mapstructure:"http_host"`
	HTTPPort            int      `mapstructure:"http_port"`
	HTTPStaticPath      string   `mapstructure:"http_static_path"`
	HTTPCookieKey       string   `mapstructure:"http_cookie_key"`
	HTTPClientTimeout   int      `mapstructure:"http_client_timeout"`
	HTTPCorsOrigins     []string `mapstructure:"http_cors_origins"`
	DatabaseDSN         string   `mapstructure:"database_dsn"`
	DatabaseAutoMigrate bool     `mapstructure:"database_auto_migrate"`
	DatabaseLogQueries  bool     `mapstructure:"database_log_queries"`
}

// Config is the root config container
//
//	export discord.token=TOKEN_TOKEN_TOKEN_TOKEN_TOKEN
//	export general.steam_key=STEAM_KEY_STEAM_KEY_STEAM_KEY
//	./gbans serve
type Config struct {
	StaticConfig
	General     ConfigGeneral     `mapstructure:"general"`
	HTTP        ConfigHTTP        `mapstructure:"http"`
	Demo        ConfigDemo        `mapstructure:"demo"`
	Filter      ConfigFilter      `mapstructure:"word_filter"`
	DB          ConfigDB          `mapstructure:"database"`
	Discord     ConfigDiscord     `mapstructure:"discord"`
	Log         ConfigLog         `mapstructure:"logging"`
	IP2Location ConfigIP2Location `mapstructure:"ip2location"`
	Debug       ConfigDebug       `mapstructure:"debug"`
	Patreon     ConfigPatreon     `mapstructure:"patreon"`
	SSH         ConfigSSH         `mapstructure:"ssh"`
	LocalStore  ConfigLocalStore  `mapstructure:"local_store"`
	Exports     ConfigExports     `mapstructure:"exports"`
	Sentry      ConfigSentry
}

func (c Config) ExtURL(obj LinkablePath) string {
	return c.ExtURLRaw(obj.Path())
}

func (c Config) ExtURLRaw(path string, args ...any) string {
	return strings.TrimRight(c.General.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}

type ConfigSSH struct {
	Enabled        bool          `mapstructure:"enabled"`
	Username       string        `mapstructure:"username"`
	Port           int           `mapstructure:"port"`
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	Password       string        `mapstructure:"password"`
	UpdateInterval time.Duration `mapstructure:"update_interval"`
	Timeout        time.Duration `mapstructure:"timeout"`
	DemoPathFmt    string        `mapstructure:"demo_path_fmt"`
}

type ConfigExports struct {
	BDEnabled      bool     `mapstructure:"bd_enabled"`
	ValveEnabled   bool     `mapstructure:"valve_enabled"`
	AuthorizedKeys []string `mapstructure:"authorized_keys"`
}

type ConfigFilter struct {
	Enabled        bool          `mapstructure:"enabled"`
	WarningTimeout time.Duration `mapstructure:"warning_timeout"`
	WarningLimit   int           `mapstructure:"warning_limit"`
	Dry            bool          `mapstructure:"dry"`
	PingDiscord    bool          `mapstructure:"ping_discord"`
	MaxWeight      int           `mapstructure:"max_weight"`
	CheckTimeout   time.Duration `mapstructure:"check_timeout"`
	MatchTimeout   time.Duration `mapstructure:"match_timeout"`
}

type ConfigLocalStore struct {
	PathRoot string `mapstructure:"path_root"`
}

type ConfigDB struct {
	DSN         string `mapstructure:"dsn"`
	AutoMigrate bool   `mapstructure:"auto_migrate"`
	LogQueries  bool   `mapstructure:"log_queries"`
}

type ConfigPatreon struct {
	Enabled             bool   `mapstructure:"enabled"`
	ClientID            string `mapstructure:"client_id"`
	ClientSecret        string `mapstructure:"client_secret"`
	CreatorAccessToken  string `mapstructure:"creator_access_token"`
	CreatorRefreshToken string `mapstructure:"creator_refresh_token"`
}

type ConfigHTTP struct {
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	StaticPath    string        `mapstructure:"static_path"`
	CookieKey     string        `mapstructure:"cookie_key"`
	ClientTimeout time.Duration `mapstructure:"client_timeout"`
	CorsOrigins   []string      `mapstructure:"cors_origins"`
}

// Addr returns the address in host:port format.
func (h ConfigHTTP) Addr() string {
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

type ConfigAction string

const (
	ActionGag  ConfigAction = "gag"
	ActionKick ConfigAction = "kick"
	ActionBan  ConfigAction = "ban"
)

type FileServeMode string

const (
	S3Mode    FileServeMode = "s3"
	LocalMode FileServeMode = "local"
)

type DemoStrategy string

const (
	DemoStrategyPctFree DemoStrategy = "pctfree"
	DemoStrategyCount   DemoStrategy = "count"
)

type ConfigGeneral struct {
	SiteName    string  `mapstructure:"site_name"`
	SteamKey    string  `mapstructure:"steam_key"`
	Owner       string  `mapstructure:"owner"`
	Mode        RunMode `mapstructure:"mode"`
	ExternalURL string  `mapstructure:"external_url"`

	FileServeMode FileServeMode `mapstructure:"file_serve_mode"`
	SrcdsLogAddr  string        `mapstructure:"srcds_log_addr"`
}

type ConfigDemo struct {
	DemoCleanupEnabled  bool         `mapstructure:"demo_cleanup_enabled"`
	DemoCleanupStrategy DemoStrategy `mapstructure:"demo_cleanup_strategy"`
	DemoCleanupMinPct   float32      `mapstructure:"demo_cleanup_min_pct"`
	DemoCleanupMount    string       `mapstructure:"demo_cleanup_mount"`
	DemoCountLimit      uint64       `mapstructure:"demo_count_limit"`
}

type ConfigDiscord struct {
	Enabled                 bool   `mapstructure:"enabled"`
	AppID                   string `mapstructure:"app_id"`
	AppSecret               string `mapstructure:"app_secret"`
	LinkID                  string `mapstructure:"link_id"`
	Token                   string `mapstructure:"token"`
	GuildID                 string `mapstructure:"guild_id"`
	LogChannelID            string `mapstructure:"log_channel_id"`
	PublicLogChannelEnable  bool   `mapstructure:"public_log_channel_enable"`
	PublicLogChannelID      string `mapstructure:"public_log_channel_id"`
	PublicMatchLogChannelID string `mapstructure:"public_match_log_channel_id"`
	ModPingRoleID           string `mapstructure:"mod_ping_role_id"`
	UnregisterOnStart       bool   `mapstructure:"unregister_on_start"`
}

type ConfigSentry struct {
	SentryDSN        string  `mapstructure:"sentry_dsn"`
	SentryDSNWeb     string  `mapstructure:"sentry_dsn_web"`
	SentryTrace      bool    `mapstructure:"sentry_trace"`
	SentrySampleRate float64 `mapstructure:"sentry_sample_rate"`
}

type ConfigLog struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type ConfigDebug struct {
	SkipOpenIDValidation bool   `mapstructure:"skip_open_id_validation"`
	AddRCONLogAddress    string `mapstructure:"add_rcon_log_address"`
}

type ConfigIP2Location struct {
	Enabled   bool   `mapstructure:"enabled"`
	CachePath string `mapstructure:"cache_path"`
	Token     string `mapstructure:"token"`
}
