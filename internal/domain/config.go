package domain

import (
	"context"
	"fmt"
	"strings"
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
	Owner               string   `mapstructure:"owner" json:"owner,omitempty"`
	ExternalURL         string   `mapstructure:"external_url" json:"external_url,omitempty"`
	HTTPHost            string   `mapstructure:"http_host" json:"http_host,omitempty"`
	HTTPPort            int      `mapstructure:"http_port" json:"http_port,omitempty"`
	HTTPStaticPath      string   `mapstructure:"http_static_path" json:"http_static_path,omitempty"`
	HTTPCookieKey       string   `mapstructure:"http_cookie_key" json:"-"`
	HTTPClientTimeout   int      `mapstructure:"http_client_timeout" json:"http_client_timeout,omitempty"`
	HTTPCorsOrigins     []string `mapstructure:"http_cors_origins" json:"http_cors_origins,omitempty"`
	DatabaseDSN         string   `mapstructure:"database_dsn" json:"-"`
	DatabaseAutoMigrate bool     `mapstructure:"database_auto_migrate" json:"database_auto_migrate,omitempty"`
	DatabaseLogQueries  bool     `mapstructure:"database_log_queries" json:"database_log_queries,omitempty"`
}

// Addr returns the address in host:port format.
func (s StaticConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.HTTPHost, s.HTTPPort)
}

// Config is the root config container
//
//	export discord.token=TOKEN_TOKEN_TOKEN_TOKEN_TOKEN
//	export general.steam_key=STEAM_KEY_STEAM_KEY_STEAM_KEY
//	./gbans serve
type Config struct {
	StaticConfig
	General     ConfigGeneral     `json:"general"`
	Demo        ConfigDemo        `json:"demo"`
	Filters     ConfigFilter      `json:"filters"`
	Discord     ConfigDiscord     `json:"discord"`
	Log         ConfigLog         `json:"log"`
	GeoLocation ConfigIP2Location `json:"geo_location"`
	Debug       ConfigDebug       `json:"debug"`
	Patreon     ConfigPatreon     `json:"patreon"`
	SSH         ConfigSSH         `json:"ssh"`
	LocalStore  ConfigLocalStore  `json:"local_store"`
	Exports     ConfigExports     `json:"exports"`
	Sentry      ConfigSentry      `json:"sentry"`
}

func (c Config) ExtURL(obj LinkablePath) string {
	return c.ExtURLRaw(obj.Path())
}

func (c Config) ExtURLRaw(path string, args ...any) string {
	return strings.TrimRight(c.StaticConfig.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}

type ConfigSSH struct {
	Enabled        bool   `json:"enabled"`
	Username       string `json:"username"`
	Port           int    `json:"port,string"`
	PrivateKeyPath string `json:"private_key_path"`
	Password       string `json:"password"`
	UpdateInterval int    `json:"update_interval,string"`
	Timeout        int    `json:"timeout,string"`
	DemoPathFmt    string `json:"demo_path_fmt"`
}

type ConfigExports struct {
	BDEnabled      bool     `json:"bd_enabled"`
	ValveEnabled   bool     `json:"valve_enabled"`
	AuthorizedKeys []string `json:"authorized_keys"`
}

type ConfigFilter struct {
	Enabled        bool `json:"enabled"`
	WarningTimeout int  `json:"warning_timeout,string"`
	WarningLimit   int  `json:"warning_limit,string"`
	Dry            bool `json:"dry"`
	PingDiscord    bool `json:"ping_discord"`
	MaxWeight      int  `json:"max_weight,string"`
	CheckTimeout   int  `json:"check_timeout,string"`
	MatchTimeout   int  `json:"match_timeout,string"`
}

type ConfigLocalStore struct {
	PathRoot string `json:"path_root"`
}

type ConfigDB struct {
	DSN         string `mapstructure:"dsn"`
	AutoMigrate bool   `mapstructure:"auto_migrate"`
	LogQueries  bool   `mapstructure:"log_queries"`
}

type ConfigPatreon struct {
	Enabled             bool   `json:"enabled"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	CreatorAccessToken  string `json:"creator_access_token"`
	CreatorRefreshToken string `json:"creator_refresh_token"`
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
	SiteName      string        `json:"site_name"`
	SteamKey      string        `json:"steam_key"`
	Mode          RunMode       `json:"mode"`
	FileServeMode FileServeMode `json:"file_serve_mode"`
	SrcdsLogAddr  string        `json:"srcds_log_addr"`
}

type ConfigDemo struct {
	DemoCleanupEnabled  bool         `json:"demo_cleanup_enabled"`
	DemoCleanupStrategy DemoStrategy `json:"demo_cleanup_strategy"`
	DemoCleanupMinPct   float32      `json:"demo_cleanup_min_pct,string"`
	DemoCleanupMount    string       `json:"demo_cleanup_mount"`
	DemoCountLimit      uint64       `json:"demo_count_limit,string"`
}

type ConfigDiscord struct {
	Enabled                 bool   `json:"enabled"`
	AppID                   string `json:"app_id"`
	AppSecret               string `json:"app_secret"`
	LinkID                  string `json:"link_id"`
	Token                   string `json:"token"`
	GuildID                 string `json:"guild_id"`
	LogChannelID            string `json:"log_channel_id"`
	PublicLogChannelEnable  bool   `json:"public_log_channel_enable"`
	PublicLogChannelID      string `json:"public_log_channel_id"`
	PublicMatchLogChannelID string `json:"public_match_log_channel_id"`
	ModPingRoleID           string `json:"mod_ping_role_id"`
	UnregisterOnStart       bool   `json:"unregister_on_start"`
}

type ConfigSentry struct {
	SentryDSN        string  `json:"sentry_dsn"`
	SentryDSNWeb     string  `json:"sentry_dsn_web"`
	SentryTrace      bool    `json:"sentry_trace"`
	SentrySampleRate float64 `json:"sentry_sample_rate,string"`
}

type ConfigLog struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type ConfigDebug struct {
	SkipOpenIDValidation bool   `json:"skip_open_id_validation"`
	AddRCONLogAddress    string `json:"add_rcon_log_address"`
}

type ConfigIP2Location struct {
	Enabled   bool   `json:"enabled"`
	CachePath string `json:"cache_path"`
	Token     string `json:"token"`
}
