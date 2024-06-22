package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type configUsecase struct {
	repository    domain.ConfigRepository
	static        domain.StaticConfig
	configMu      sync.RWMutex
	currentConfig domain.Config
}

func NewConfigUsecase(static domain.StaticConfig, repository domain.ConfigRepository) domain.ConfigUsecase {
	return &configUsecase{static: static, repository: repository}
}

func (c *configUsecase) Init(ctx context.Context) error {
	return c.repository.Init(ctx)
}

func (c *configUsecase) Write(ctx context.Context, config domain.Config) error {
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

func (c *configUsecase) ExtURL(obj domain.LinkablePath) string {
	return c.Config().ExtURL(obj)
}

func (c *configUsecase) ExtURLRaw(path string, args ...any) string {
	return c.Config().ExtURLRaw(path, args...)
}

func (c *configUsecase) Config() domain.Config {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	return c.currentConfig
}

func (c *configUsecase) Reload(ctx context.Context) error {
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

func ReadStaticConfig() (domain.StaticConfig, error) {
	setDefaultConfigValues()

	var config domain.StaticConfig
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

func applyGlobalConfig(config domain.Config) error {
	gin.SetMode(config.General.Mode.String())

	if errSteam := steamid.SetKey(config.SteamKey); errSteam != nil {
		return errors.Join(errSteam, domain.ErrSteamAPIKey)
	}

	if errSteamWeb := steamweb.SetKey(config.SteamKey); errSteamWeb != nil {
		return errors.Join(errSteamWeb, domain.ErrSteamAPIKey)
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
