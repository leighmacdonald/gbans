package config

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type ConfigHandler struct {
	config *Configuration
}

func NewConfigHandler(engine *gin.Engine, cu *Configuration, authUC httphelper.Authenticator, version string) {
	handler := ConfigHandler{config: cu}
	engine.GET("/api/info", handler.onAppInfo(version))
	engine.GET("/api/changelog", handler.onChangelog())

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(authUC.Middleware(permission.PAdmin))
		admin.GET("/api/config", handler.onAPIGetConfig())
		admin.PUT("/api/config", handler.onAPIPutConfig())
	}
}

func (c ConfigHandler) onAPIGetConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, c.config.Config())
	}
}

func (c ConfigHandler) onAPIPutConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Config
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := c.config.Write(ctx, req); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Failed to write new config"))

			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func (c ConfigHandler) onAppInfo(version string) gin.HandlerFunc {
	type appInfo struct {
		SiteName           string `json:"site_name"`
		AssetURL           string `json:"asset_url"`
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

	return func(ctx *gin.Context) {
		conf := c.config.Config()

		ctx.JSON(http.StatusOK, appInfo{
			SiteName:           conf.General.SiteName,
			AssetURL:           conf.General.AssetURL,
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

func (c ConfigHandler) onChangelog() gin.HandlerFunc {
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
