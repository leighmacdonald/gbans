package domain

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/leighmacdonald/gbans/pkg/log"
)

type LinkablePath interface {
	// Path returns the HTTP path that is represented by the instance.
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
	ExtURLInstance(obj LinkablePath) *url.URL
	ExtURLRaw(path string, args ...any) string
	Reload(ctx context.Context) error
	Write(ctx context.Context, config Config) error
	Init(ctx context.Context) error
}

// StaticConfig defines non-dynamic config values that cannot be changed during runtime. These
// are loaded via the config file.
type StaticConfig struct {
	Owner               string   `mapstructure:"owner" json:"-"`
	SteamKey            string   `mapstructure:"steam_key" json:"-"`
	ExternalURL         string   `mapstructure:"external_url" json:"-"`
	HTTPHost            string   `mapstructure:"http_host" json:"-"`
	HTTPPort            uint16   `mapstructure:"http_port" json:"-"`
	HTTPStaticPath      string   `mapstructure:"http_static_path" json:"-"`
	HTTPCookieKey       string   `mapstructure:"http_cookie_key" json:"-"`
	HTTPClientTimeout   int      `mapstructure:"http_client_timeout" json:"-"`
	HTTPCORSEnabled     bool     `mapstructure:"http_cors_enabled"`
	HTTPCorsOrigins     []string `mapstructure:"http_cors_origins" json:"-"`
	DatabaseDSN         string   `mapstructure:"database_dsn" json:"-"`
	DatabaseAutoMigrate bool     `mapstructure:"database_auto_migrate" json:"-"`
	DatabaseLogQueries  bool     `mapstructure:"database_log_queries" json:"-"`
	PrometheusEnabled   bool     `mapstructure:"prometheus_enabled"`
	PProfEnabled        bool     `mapstructure:"pprof_enabled"`
}

// Addr returns the address in host:port format.
func (s StaticConfig) Addr() string {
	return net.JoinHostPort(s.HTTPHost, strconv.Itoa(int(s.HTTPPort)))
}

type ConfigAnticheat struct {
	Enabled               bool         `mapstructure:"enabled" json:"enabled"`
	Action                ConfigAction `mapstructure:"action" json:"action"`
	Duration              int          `mapstructure:"duration" json:"duration"`
	MaxAimSnap            int          `mapstructure:"max_aim_snap" json:"max_aim_snap"`
	MaxPsilent            int          `mapstructure:"max_psilent" json:"max_psilent"`
	MaxBhop               int          `mapstructure:"max_bhop" json:"max_bhop"`
	MaxFakeAng            int          `mapstructure:"max_fake_ang" json:"max_fake_ang"`
	MaxCmdNum             int          `mapstructure:"max_cmd_num" json:"max_cmd_num"`
	MaxTooManyConnections int          `mapstructure:"max_too_many_connections" json:"max_too_many_connections"`
	MaxCheatCvar          int          `mapstructure:"max_cheat_cvar" json:"max_cheat_cvar"`
	MaxOOBVar             int          `mapstructure:"max_oob_var" json:"max_oob_var"`
	MaxInvalidUserCmd     int          `mapstructure:"max_invalid_user_cmd" json:"max_invalid_user_cmd"`
}

// Config is the root config container.
type Config struct {
	StaticConfig
	General     ConfigGeneral     `json:"general"`
	Demo        ConfigDemo        `json:"demo"`
	Filters     ConfigFilter      `json:"filters"`
	Discord     ConfigDiscord     `json:"discord"`
	Clientprefs ConfigClientprefs `json:"clientprefs"`
	Log         ConfigLog         `json:"log"`
	GeoLocation ConfigIP2Location `json:"geo_location"`
	Debug       ConfigDebug       `json:"debug"`
	Patreon     ConfigPatreon     `json:"patreon"`
	SSH         ConfigSSH         `json:"ssh"`
	LocalStore  ConfigLocalStore  `json:"local_store"`
	Exports     ConfigExports     `json:"exports"`
	Anticheat   ConfigAnticheat   `json:"anticheat"`
}

func (c Config) ExtURLInstance(obj LinkablePath) *url.URL {
	urlObj, err := url.Parse(c.ExtURLRaw(obj.Path()))
	if err != nil {
		slog.Error("Failed to parse URL", slog.String("url", c.ExtURLRaw(obj.Path())), log.ErrAttr(err))
	}

	return urlObj
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
	StacPathFmt    string `json:"stac_path_fmt"`
	// TODO configurable handling of host keys
}

type ConfigExports struct {
	BDEnabled      bool   `json:"bd_enabled"`
	ValveEnabled   bool   `json:"valve_enabled"`
	AuthorizedKeys string `json:"authorized_keys"`
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

type ConfigClientprefs struct {
	CenterProjectiles bool `mapstructure:"center_projectiles"`
}

type ConfigPatreon struct {
	Enabled             bool   `json:"enabled"`
	IntegrationsEnabled bool   `json:"integrations_enabled"`
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
	SiteName           string        `json:"site_name"`
	Mode               RunMode       `json:"mode"`
	FileServeMode      FileServeMode `json:"file_serve_mode"`
	SrcdsLogAddr       string        `json:"srcds_log_addr"`
	AssetURL           string        `json:"asset_url"`
	DefaultRoute       string        `json:"default_route"`
	NewsEnabled        bool          `json:"news_enabled"`
	ForumsEnabled      bool          `json:"forums_enabled"`
	ContestsEnabled    bool          `json:"contests_enabled"`
	WikiEnabled        bool          `json:"wiki_enabled"`
	StatsEnabled       bool          `json:"stats_enabled"`
	ServersEnabled     bool          `json:"servers_enabled"`
	ReportsEnabled     bool          `json:"reports_enabled"`
	ChatlogsEnabled    bool          `json:"chatlogs_enabled"`
	DemosEnabled       bool          `json:"demos_enabled"`
	SpeedrunsEnabled   bool          `json:"speedruns_enabled"`
	PlayerqueueEnabled bool          `json:"playerqueue_enabled"`
}

type ConfigDemo struct {
	DemoCleanupEnabled  bool         `json:"demo_cleanup_enabled"`
	DemoCleanupStrategy DemoStrategy `json:"demo_cleanup_strategy"`
	DemoCleanupMinPct   float32      `json:"demo_cleanup_min_pct,string"`
	DemoCleanupMount    string       `json:"demo_cleanup_mount"`
	DemoCountLimit      uint64       `json:"demo_count_limit,string"`
	DemoParserURL       string       `json:"demo_parser_url"`
}

type ConfigDiscord struct {
	Enabled                 bool   `json:"enabled"`
	BotEnabled              bool   `json:"bot_enabled"`
	IntegrationsEnabled     bool   `json:"integrations_enabled"`
	AppID                   string `json:"app_id"`
	AppSecret               string `json:"app_secret"`
	LinkID                  string `json:"link_id"`
	Token                   string `json:"token"`
	GuildID                 string `json:"guild_id"`
	LogChannelID            string `json:"log_channel_id"`
	PublicLogChannelEnable  bool   `json:"public_log_channel_enable"`
	PublicLogChannelID      string `json:"public_log_channel_id"`
	PublicMatchLogChannelID string `json:"public_match_log_channel_id"`
	VoteLogChannelID        string `json:"vote_log_channel_id"`
	AppealLogChannelID      string `json:"appeal_log_channel_id"`
	BanLogChannelID         string `json:"ban_log_channel_id"`
	ForumLogChannelID       string `json:"forum_log_channel_id"`
	WordFilterLogChannelID  string `json:"word_filter_log_channel_id"`
	KickLogChannelID        string `json:"kick_log_channel_id"`
	PlayerqueueChannelID    string `json:"playerqueue_channel_id"`
	ModPingRoleID           string `json:"mod_ping_role_id"`
	AnticheatChannelID      string `json:"anticheat_channel_id"`
}

type ConfigLog struct {
	Level log.Level `json:"level"`
	// If set to a non-empty path, logs will also be written to the log file.
	File string `json:"file"`
	// Enable using the sloggin library for logging HTTP requests
	HTTPEnabled bool `json:"http_enabled"`
	// Enable support for OpenTelemetry by adding span/trace IDs
	HTTPOtelEnabled bool `json:"http_otel_enabled"`
	// Log level to use for http requests
	HTTPLevel log.Level `json:"http_level"`
}

type ConfigDebug struct {
	SkipOpenIDValidation bool `json:"skip_open_id_validation"`
	// Will send the `logaddress_add <ip>:<port>` rcon command to all enabled servers so that
	// you can forward them to yourself for testing. This does not remove any existing entries.
	AddRCONLogAddress string `json:"add_rcon_log_address"`
}

type ConfigIP2Location struct {
	Enabled   bool   `json:"enabled"`
	CachePath string `json:"cache_path"`
	Token     string `json:"token"`
}
