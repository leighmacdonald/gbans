package domain

import (
	"fmt"
	"strings"
	"time"
)

type LinkablePath interface {
	Path() string
}

type ConfigRepository interface {
	Config() Config
	Read(noFileOk bool) error
}

type ConfigUsecase interface {
	Read(noFileOk bool) error
	Config() Config
	ExtURL(obj LinkablePath) string
	ExtURLRaw(path string, args ...any) string
}

// Config is the root config container
//
//	export discord.token=TOKEN_TOKEN_TOKEN_TOKEN_TOKEN
//	export general.steam_key=STEAM_KEY_STEAM_KEY_STEAM_KEY
//	./gbans serve
type Config struct {
	General     ConfigGeneral     `mapstructure:"general"`
	HTTP        ConfigHTTP        `mapstructure:"http"`
	Filter      ConfigFilter      `mapstructure:"word_filter"`
	DB          ConfigDB          `mapstructure:"database"`
	Discord     ConfigDiscord     `mapstructure:"discord"`
	Log         ConfigLog         `mapstructure:"logging"`
	IP2Location ConfigIP2Location `mapstructure:"ip2location"`
	Debug       ConfigDebug       `mapstructure:"debug"`
	Patreon     ConfigPatreon     `mapstructure:"patreon"`
	S3          ConfigS3          `mapstructure:"s3"`
	SSH         ConfigSSH         `mapstructure:"ssh"`
	LocalStore  ConfigLocalStore  `mapstructure:"local_store"`
	Exports     ConfigExports     `mapstructure:"exports"`
}

func (c Config) ExtURL(obj LinkablePath) string {
	return c.ExtURLRaw(obj.Path())
}

func (c Config) ExtURLRaw(path string, args ...any) string {
	return strings.TrimRight(c.General.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}

type ConfigSSH struct {
	Username       string        `mapstructure:"username"`
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	Password       string        `mapstructure:"password"`
	UpdateInterval time.Duration `mapstructure:"update_interval"`
	Timeout        time.Duration `mapstructure:"timeout"`
}

type ConfigExports struct {
	BDEnabled      bool     `mapstructure:"bd_enabled"`
	ValveEnabled   bool     `mapstructure:"valve_enabled"`
	AuthorizedKeys []string `mapstructure:"authorized_keys"`
}

type ConfigFilter struct {
	Enabled      bool          `mapstructure:"enabled"`
	Dry          bool          `mapstructure:"dry"`
	PingDiscord  bool          `mapstructure:"ping_discord"`
	MaxWeight    int           `mapstructure:"max_weight"`
	CheckTimeout time.Duration `mapstructure:"check_timeout"`
	MatchTimeout time.Duration `mapstructure:"match_timeout"`
}

type ConfigLocalStore struct {
	PathRoot string `mapstructure:"path_root"`
}

type ConfigS3Store struct {
	Enabled     bool   `mapstructure:"enabled"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	Endpoint    string `mapstructure:"endpoint"`
	ExternalURL string `mapstructure:"external_url"`
	Region      string `mapstructure:"region"`
	SSL         bool   `mapstructure:"ssl"`
	BucketMedia string `mapstructure:"bucket_media"`
	BucketDemo  string `mapstructure:"bucket_demo"`
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
	TLS           bool          `mapstructure:"tls"`
	TLSAuto       bool          `mapstructure:"tls_auto"`
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

type ConfigGeneral struct {
	SiteName                     string        `mapstructure:"site_name"`
	SteamKey                     string        `mapstructure:"steam_key"`
	Owner                        string        `mapstructure:"owner"`
	Mode                         RunMode       `mapstructure:"mode"`
	WarningTimeout               time.Duration `mapstructure:"warning_timeout"`
	WarningLimit                 int           `mapstructure:"warning_limit"`
	UseUTC                       bool          `mapstructure:"use_utc"`
	ServerStatusUpdateFreq       time.Duration `mapstructure:"server_status_update_freq"`
	MasterServerStatusUpdateFreq time.Duration `mapstructure:"master_server_status_update_freq"`
	DefaultMaps                  []string      `mapstructure:"default_maps"`
	ExternalURL                  string        `mapstructure:"external_url"`
	DemoCleanupEnabled           bool          `mapstructure:"demo_cleanup_enabled"`
	DemoCountLimit               uint64        `mapstructure:"demo_count_limit"`
	FileServeMode                FileServeMode `mapstructure:"file_serve_mode"`
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

type ConfigLog struct {
	Level            string  `mapstructure:"level"`
	File             string  `mapstructure:"file"`
	ReportCaller     bool    `mapstructure:"report_caller"`
	FullTimestamp    bool    `mapstructure:"full_timestamp"`
	SrcdsLogAddr     string  `mapstructure:"srcds_log_addr"`
	SentryDSN        string  `mapstructure:"sentry_dsn"`
	SentryDSNWeb     string  `mapstructure:"sentry_dsn_web"`
	SentryTrace      bool    `mapstructure:"sentry_trace"`
	SentrySampleRate float64 `mapstructure:"sentry_sample_rate"`
}

type ConfigDebug struct {
	UpdateSRCDSLogSecrets   bool   `mapstructure:"update_srcds_log_secrets"`
	SkipOpenIDValidation    bool   `mapstructure:"skip_open_id_validation"`
	WriteUnhandledLogEvents bool   `mapstructure:"write_unhandled_log_events"`
	AddRCONLogAddress       string `mapstructure:"add_rcon_log_address"`
}

type ConfigIP2Location struct {
	Enabled      bool   `mapstructure:"enabled"`
	CachePath    string `mapstructure:"cache_path"`
	Token        string `mapstructure:"token"`
	ASNEnabled   bool   `mapstructure:"asn_enabled"`
	IPEnabled    bool   `mapstructure:"ip_enabled"`
	ProxyEnabled bool   `mapstructure:"proxy_enabled"`
}
