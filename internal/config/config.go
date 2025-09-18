package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type LinkablePath interface {
	// Path returns the HTTP path that is represented by the instance.
	Path() string
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
	HTTPCORSEnabled     bool     `mapstructure:"http_cors_enabled"  json:"-"`
	HTTPCorsOrigins     []string `mapstructure:"http_cors_origins" json:"-"`
	DatabaseDSN         string   `mapstructure:"database_dsn" json:"-"`
	DatabaseAutoMigrate bool     `mapstructure:"database_auto_migrate" json:"-"`
	DatabaseLogQueries  bool     `mapstructure:"database_log_queries" json:"-"`
	PrometheusEnabled   bool     `mapstructure:"prometheus_enabled" json:"-"`
	PProfEnabled        bool     `mapstructure:"pprof_enabled" json:"-"`
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
	General     General           `json:"general"`
	Demo        Demo              `json:"demo"`
	Filters     ConfigFilter      `json:"filters"`
	Discord     Discord           `json:"discord"`
	Clientprefs ConfigClientprefs `json:"clientprefs"`
	Log         Log               `json:"log"`
	GeoLocation IP2Location       `json:"geo_location"`
	Debug       Debug             `json:"debug"`
	Patreon     ConfigPatreon     `json:"patreon"`
	SSH         ConfigSSH         `json:"ssh"`
	Network     ConfigNetwork     `json:"network"`
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
	return strings.TrimRight(c.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}

type ConfigNetwork struct {
	SDREnabled    bool   `mapstructure:"sdr_enabled" json:"sdr_enabled"`
	SDRDNSEnabled bool   `mapstructure:"sdr_dns_enabled" json:"sdr_dns_enabled"` // nolint:tagliatelle
	CFKey         string `mapstructure:"cf_key" json:"cf_key"`
	CFEmail       string `mapstructure:"cf_email" json:"cf_email"`
	CFZoneID      string `mapstructure:"cf_zone_id" json:"cf_zone_id"`
}

type ConfigSSH struct {
	Enabled        bool   `json:"enabled"`
	Username       string `json:"username"`
	Port           int    `json:"port"`
	PrivateKeyPath string `json:"private_key_path"`
	Password       string `json:"password"`
	UpdateInterval int    `json:"update_interval"`
	Timeout        int    `json:"timeout"`
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
	WarningTimeout int  `json:"warning_timeout"`
	WarningLimit   int  `json:"warning_limit"`
	Dry            bool `json:"dry"`
	PingDiscord    bool `json:"ping_discord"`
	MaxWeight      int  `json:"max_weight"`
	CheckTimeout   int  `json:"check_timeout"`
	MatchTimeout   int  `json:"match_timeout"`
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

type General struct {
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

type Demo struct {
	DemoCleanupEnabled  bool         `json:"demo_cleanup_enabled"`
	DemoCleanupStrategy DemoStrategy `json:"demo_cleanup_strategy"`
	DemoCleanupMinPct   float32      `json:"demo_cleanup_min_pct"`
	DemoCleanupMount    string       `json:"demo_cleanup_mount"`
	DemoCountLimit      uint64       `json:"demo_count_limit"`
	DemoParserURL       string       `json:"demo_parser_url"`
}

type Discord struct {
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

type Log struct {
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

type Debug struct {
	SkipOpenIDValidation bool `json:"skip_open_id_validation"`
	// Will send the `logaddress_add <ip>:<port>` rcon command to all enabled servers so that
	// you can forward them to yourself for testing. This does not remove any existing entries.
	AddRCONLogAddress string `json:"add_rcon_log_address"`
}

type IP2Location struct {
	Enabled   bool   `json:"enabled"`
	CachePath string `json:"cache_path"`
	Token     string `json:"token"`
}

type Configuration struct {
	repository    *Repository
	static        StaticConfig
	configMu      sync.RWMutex
	currentConfig Config
}

func NewConfiguration(static StaticConfig, repository *Repository) *Configuration {
	return &Configuration{static: static, repository: repository}
}

func (c *Configuration) Init(ctx context.Context) error {
	return c.repository.Init(ctx)
}

func (c *Configuration) Write(ctx context.Context, config Config) error {
	if err := c.repository.Write(ctx, config); err != nil {
		slog.Error("Failed to write new config", log.ErrAttr(err))

		return err
	}

	if errReload := c.Reload(ctx); errReload != nil {
		slog.Error("Failed to reload config", log.ErrAttr(errReload))

		return errReload
	}

	return nil
}

func (c *Configuration) ExtURLInstance(obj LinkablePath) *url.URL {
	return c.Config().ExtURLInstance(obj)
}

func (c *Configuration) ExtURL(obj LinkablePath) string {
	return c.Config().ExtURL(obj)
}

func (c *Configuration) ExtURLRaw(path string, args ...any) string {
	return c.Config().ExtURLRaw(path, args...)
}

func (c *Configuration) Config() Config {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	return c.currentConfig
}

func (c *Configuration) Reload(ctx context.Context) error {
	config, errConfig := c.repository.Read(ctx)
	if errConfig != nil {
		return errConfig
	}

	config.StaticConfig = c.static

	c.configMu.Lock()
	c.currentConfig = config
	c.configMu.Unlock()

	if err := applyGlobalConfig(config); err != nil {
		return err
	}

	return nil
}

func ReadStaticConfig() (StaticConfig, error) {
	setDefaultConfigValues()

	var config StaticConfig
	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return config, errors.Join(errReadConfig, domain.ErrReadConfig)
	}

	if errUnmarshal := viper.Unmarshal(&config, viper.DecodeHook(mapstructure.DecodeHookFunc(decodeDuration()))); errUnmarshal != nil {
		return config, errors.Join(errUnmarshal, domain.ErrFormatConfig)
	}

	if strings.HasPrefix(config.DatabaseDSN, "pgx://") {
		config.DatabaseDSN = strings.Replace(config.DatabaseDSN, "pgx://", "postgres://", 1)
	}

	if _, errParse := url.Parse(config.DatabaseDSN); errParse != nil {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "database_dsn")
	}

	if len(config.SteamKey) != 32 {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "steam_key")
	}

	ownerSID := steamid.New(config.Owner)
	if !ownerSID.Valid() {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "owner")
	}

	if config.ExternalURL == "" {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "external_url")
	}

	if parsed, errParse := url.Parse(config.ExternalURL); errParse != nil || parsed.Host == "" {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "external_url")
	}

	if !slices.Contains(config.HTTPCorsOrigins, config.ExternalURL) {
		config.HTTPCorsOrigins = append(config.HTTPCorsOrigins, config.ExternalURL)
	}

	if len(config.HTTPCookieKey) < 10 {
		return config, fmt.Errorf("%w: %s", domain.ErrInvalidConfig, "http_cookie_key")
	}

	return config, nil
}

func applyGlobalConfig(config Config) error {
	gin.SetMode(config.General.Mode.String())

	if errSteam := steamid.SetKey(config.SteamKey); errSteam != nil {
		return errors.Join(errSteam, domain.ErrSteamAPIKey)
	}

	return nil
}

type GithubRelease struct {
	URL             string    `json:"url"`
	HTMLUrl         string    `json:"html_url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	ID              int       `json:"id"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Author          struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLUrl           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	Assets []struct {
		URL                string    `json:"url"`
		BrowserDownloadURL string    `json:"browser_download_url"`
		ID                 int       `json:"id"`
		NodeID             string    `json:"node_id"`
		Name               string    `json:"name"`
		Label              string    `json:"label"`
		State              string    `json:"state"`
		ContentType        string    `json:"content_type"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		Uploader           struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLUrl           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
	} `json:"assets"`
}

func getGithubReleases(ctx context.Context) ([]GithubRelease, error) {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/leighmacdonald/gbans/releases", nil)
	if errReq != nil {
		return nil, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	client := httphelper.NewHTTPClient()

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close github releases body", log.ErrAttr(err))
		}
	}()

	var releases []GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, errors.Join(err, domain.ErrRequestDecode)
	}

	return releases, nil
}

// decodeDuration automatically parses the string duration type (1s,1m,1h,etc.) into a real time.Duration type.
func decodeDuration() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, target reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		// t.TypeOf doesn't seem to work with time.Duration, so just grug it.
		if !strings.HasSuffix(target.String(), "Duration") && !strings.HasSuffix(target.String(), "Freq") {
			return data, nil
		}

		durString, ok := data.(string)
		if !ok {
			return nil, domain.ErrDecodeDuration
		}

		duration, errDuration := datetime.ParseUserStringDuration(durString)
		if errDuration != nil {
			return nil, errors.Join(errDuration, fmt.Errorf("%w: %s", domain.ErrDecodeDuration, target.String()))
		}

		return duration, nil
	}
}

func setDefaultConfigValues() {
	if home, errHomeDir := homedir.Dir(); errHomeDir != nil {
		viper.AddConfigPath(home)
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("gbans")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("gbans")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	defaultConfig := map[string]any{
		"owner":                 "",
		"external_url":          "",
		"steam_key":             "",
		"http_host":             "127.0.0.1",
		"http_port":             6006,
		"http_static_path":      "frontend/dist",
		"http_cookie_key":       stringutil.SecureRandomString(32),
		"http_client_timeout":   "10",
		"http_cors_enabled":     true,
		"http_cors_origins":     []string{"http://gbans.localhost"},
		"database_dsn":          "postgresql://gbans:gbans@localhost/gbans",
		"database_auto_migrate": true,
		"database_log_queries":  false,
		"prometheus_enabled":    false,
		"pprof_enabled":         false,
	}

	for configKey, value := range defaultConfig {
		viper.SetDefault(configKey, value)
	}

	if errWriteConfig := viper.SafeWriteConfig(); errWriteConfig != nil {
		return
	}
}
