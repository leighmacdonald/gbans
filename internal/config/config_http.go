package config

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type httpHandler struct {
	*Configuration
}

func NewHandler(engine *gin.Engine, authenticator httphelper.Authenticator, cu *Configuration, version string) {
	handler := httpHandler{cu}

	engine.GET("/api/info", handler.onAppInfo(version))
	engine.GET("/api/changelog", handler.onChangelog())

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(authenticator.Middleware(permission.Admin))
		admin.GET("/api/config", handler.onAPIGetConfig())
		admin.PUT("/api/config", handler.onAPIPutConfig())
	}
}

func (c httpHandler) onAPIGetConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, c.Config())
	}
}

func (c httpHandler) onAPIPutConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindJSON[Config](ctx)
		if !ok {
			return
		}

		if errSave := c.Write(ctx, req); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Failed to write new config"))

			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}

type AppInfo struct {
	SiteName           string `json:"site_name"`
	SiteDescription    string `json:"site_description"`
	AssetURL           string `json:"asset_url"`
	Favicon            string `json:"favicon"`
	LinkID             string `json:"link_id"`
	AppVersion         string `json:"app_version"`
	DocumentPolicy     string `json:"document_policy"`
	PatreonClientID    string `json:"patreon_client_id"`
	DiscordClientID    string `json:"discord_client_id"`
	DiscordEnabled     bool   `json:"discord_enabled"`
	PatreonEnabled     bool   `json:"patreon_enabled"`
	DefaultRoute       string `json:"default_route"`
	NewsEnabled        bool   `json:"news_enabled"`
	ForumsEnabled      bool   `json:"forums_enabled"`
	ContestsEnabled    bool   `json:"contests_enabled"`
	WikiEnabled        bool   `json:"wiki_enabled"`
	StatsEnabled       bool   `json:"stats_enabled"`
	ServersEnabled     bool   `json:"servers_enabled"`
	ReportsEnabled     bool   `json:"reports_enabled"`
	ChatlogsEnabled    bool   `json:"chatlogs_enabled"`
	DemosEnabled       bool   `json:"demos_enabled"`
	SpeedrunsEnabled   bool   `json:"speedruns_enabled"`
	PlayerqueueEnabled bool   `json:"playerqueue_enabled"`
}

func (c httpHandler) onAppInfo(version string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		conf := c.Config()

		ctx.JSON(http.StatusOK, AppInfo{
			SiteName:           conf.General.SiteName,
			AssetURL:           conf.General.AssetURL,
			Favicon:            conf.General.FaviconURL(),
			LinkID:             conf.Discord.LinkID,
			AppVersion:         version,
			DocumentPolicy:     "",
			PatreonClientID:    conf.Patreon.ClientID,
			DiscordClientID:    conf.Discord.AppID,
			DiscordEnabled:     conf.Discord.IntegrationsEnabled && conf.Discord.Enabled,
			PatreonEnabled:     conf.Patreon.IntegrationsEnabled && conf.Patreon.Enabled,
			DefaultRoute:       conf.General.DefaultRoute,
			NewsEnabled:        conf.General.NewsEnabled,
			ForumsEnabled:      conf.General.ForumsEnabled,
			ContestsEnabled:    conf.General.ContestsEnabled,
			WikiEnabled:        conf.General.WikiEnabled,
			StatsEnabled:       conf.General.StatsEnabled,
			ServersEnabled:     conf.General.ServersEnabled,
			ReportsEnabled:     conf.General.ReportsEnabled,
			ChatlogsEnabled:    conf.General.ChatlogsEnabled,
			DemosEnabled:       conf.General.DemosEnabled,
			SpeedrunsEnabled:   conf.General.SpeedrunsEnabled,
			PlayerqueueEnabled: conf.General.PlayerqueueEnabled,
		})
	}
}

func (c httpHandler) onChangelog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		releases, err := getGithubReleases(ctx)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal),
				"Failed to load changelog from github"))

			return
		}

		ctx.JSON(http.StatusOK, releases)
	}
}
