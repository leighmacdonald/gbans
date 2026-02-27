package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-viper/mapstructure/v2"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	ErrInvalidConfig  = errors.New("invalid config value")
	ErrSteamAPIKey    = errors.New("failed to set steam api key")
	ErrReadConfig     = errors.New("failed to read config file")
	ErrFormatConfig   = errors.New("config file format invalid")
	ErrDecodeDuration = errors.New("failed to decode duration")
)

// Static defines non-dynamic config values that cannot be changed during runtime. These
// are loaded via the config file.
type Static struct {
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
func (s Static) Addr() string {
	return net.JoinHostPort(s.HTTPHost, strconv.Itoa(int(s.HTTPPort)))
}

// Config is the root config container.
type Config struct {
	// Loaded from the config file.
	Static

	// General config opts.
	General General `json:"general"`
	Debug   Debug   `json:"debug"`

	// Package configs.
	Demo        servers.DemoConfig `json:"demo"`
	Filters     chat.Config        `json:"filters"`
	Discord     discord.Config     `json:"discord"`
	Clientprefs sourcemod.Config   `json:"clientprefs"`
	Log         log.Config         `json:"log"`
	GeoLocation ip2location.Config `json:"geo_location"`
	Patreon     patreon.Config     `json:"patreon"`
	SSH         scp.Config         `json:"ssh"`
	Network     network.Config     `json:"network"`
	LocalStore  asset.Config       `json:"local_store"`
	Exports     ban.Config         `json:"exports"`
	Anticheat   anticheat.Config   `json:"anticheat"`
}

func (c Config) ExtURLRaw(path string, args ...any) string {
	return strings.TrimRight(c.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
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

type FileServeMode string

const (
	LocalMode FileServeMode = "local"
)

type General struct {
	SiteName           string        `json:"site_name"`
	SiteDescription    string        `json:"site_description"`
	Mode               RunMode       `json:"mode"`
	FileServeMode      FileServeMode `json:"file_serve_mode"`
	SrcdsLogAddr       string        `json:"srcds_log_addr"`
	AssetURL           string        `json:"asset_url"`
	Favicon            string        `json:"favicon"`
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

func (c General) FaviconURL() string {
	if c.Favicon == "" {
		return ""
	}

	return c.AssetURL + c.Favicon
}

type Debug struct {
	SkipOpenIDValidation bool `json:"skip_open_id_validation"`
	// Will send the `logaddress_add <ip>:<port>` rcon command to all enabled servers so that
	// you can forward them to yourself for testing. This does not remove any existing entries.
	AddRCONLogAddress string `json:"add_rcon_log_address"`
}

type Configuration struct {
	repository    Repo
	static        Static
	configMu      sync.RWMutex
	currentConfig Config
}

func NewConfiguration(static Static, repository Repo) *Configuration {
	return &Configuration{static: static, repository: repository}
}

func (c *Configuration) Init(ctx context.Context) error {
	return c.repository.Init(ctx)
}

func (c *Configuration) Write(ctx context.Context, config Config) error {
	if err := c.repository.Write(ctx, config); err != nil {
		slog.Error("Failed to write new config", slog.String("error", err.Error()))

		return err
	}

	if errReload := c.Reload(ctx); errReload != nil {
		slog.Error("Failed to reload config", slog.String("error", errReload.Error()))

		return errReload
	}

	return nil
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

	config.Static = c.static

	// Update the global base url.
	link.BaseURL = config.ExternalURL

	c.configMu.Lock()
	c.currentConfig = config
	c.configMu.Unlock()

	if err := applyGlobalConfig(config); err != nil {
		return err
	}

	return nil
}

func ReadStaticConfig() (Static, error) {
	setDefaultConfigValues()

	var config Static
	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return config, errors.Join(errReadConfig, ErrReadConfig)
	}

	if errUnmarshal := viper.Unmarshal(&config, viper.DecodeHook(mapstructure.DecodeHookFunc(decodeDuration()))); errUnmarshal != nil {
		return config, errors.Join(errUnmarshal, ErrFormatConfig)
	}

	if strings.HasPrefix(config.DatabaseDSN, "pgx://") {
		config.DatabaseDSN = strings.Replace(config.DatabaseDSN, "pgx://", "postgres://", 1)
	}

	if _, errParse := url.Parse(config.DatabaseDSN); errParse != nil {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "database_dsn")
	}

	if len(config.SteamKey) != 32 {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "steam_key")
	}

	ownerSID := steamid.New(config.Owner)
	if !ownerSID.Valid() {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "owner")
	}

	if config.ExternalURL == "" {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "external_url")
	}

	if parsed, errParse := url.Parse(config.ExternalURL); errParse != nil || parsed.Host == "" {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "external_url")
	}

	if !slices.Contains(config.HTTPCorsOrigins, config.ExternalURL) {
		config.HTTPCorsOrigins = append(config.HTTPCorsOrigins, config.ExternalURL)
	}

	if len(config.HTTPCookieKey) < 10 {
		return config, fmt.Errorf("%w: %s", ErrInvalidConfig, "http_cookie_key")
	}

	return config, nil
}

func applyGlobalConfig(config Config) error {
	gin.SetMode(config.General.Mode.String())

	if errSteam := steamid.SetKey(config.SteamKey); errSteam != nil {
		return errors.Join(errSteam, ErrSteamAPIKey)
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

var ErrGithubRelease = errors.New("failed to load github release")

func getGithubReleases(ctx context.Context) ([]GithubRelease, error) {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/leighmacdonald/gbans/releases", nil)
	if errReq != nil {
		return nil, fmt.Errorf("%w: %w", ErrGithubRelease, errReq)
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-Github-Api-Version", "2022-11-28")

	client := &http.Client{}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, fmt.Errorf("%w: %w", ErrGithubRelease, errResp)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close github releases body", slog.String("error", err.Error()))
		}
	}()

	releases, err := json.Decode[[]GithubRelease](resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGithubRelease, err)
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
			return nil, ErrDecodeDuration
		}

		duration, errDuration := datetime.ParseUserStringDuration(durString)
		if errDuration != nil {
			return nil, errors.Join(errDuration, fmt.Errorf("%w: %s", ErrDecodeDuration, target.String()))
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

func NewMemConfigRepository(conf Config) *MemConfigRepository {
	return &MemConfigRepository{config: conf}
}

type MemConfigRepository struct {
	config Config
	mutex  *sync.RWMutex
}

func (m *MemConfigRepository) Init(_ context.Context) error { return nil }

func (m *MemConfigRepository) Config() Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.config
}

func (m *MemConfigRepository) Read(_ context.Context) (Config, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.config, nil
}

func (m *MemConfigRepository) Write(_ context.Context, config Config) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = config

	return nil
}
