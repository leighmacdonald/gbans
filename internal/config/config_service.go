package config

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type configHandler struct {
	cu   domain.ConfigUsecase
	auth domain.AuthUsecase
}

func NewConfigHandler(engine *gin.Engine, cu domain.ConfigUsecase, auth domain.AuthUsecase, version domain.BuildInfo) {
	handler := configHandler{cu: cu, auth: auth}
	engine.GET("/api/info", handler.onAppInfo(version))
	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.AuthMiddleware(domain.PAdmin))
		admin.GET("/api/config", handler.onAPIGetConfig())
		admin.PUT("/api/config", handler.onAPIPutConfig())
	}
}

func (c configHandler) onAPIGetConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, c.cu.Config())
	}
}

func (c configHandler) onAPIPutConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.Config
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := c.cu.Write(ctx, req); errSave != nil {
			slog.Error("Failed to save new config", log.ErrAttr(errSave))
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func (c configHandler) onAppInfo(buildInfo domain.BuildInfo) gin.HandlerFunc {
	type appInfo struct {
		SiteName        string `json:"site_name"`
		AssetURL        string `json:"asset_url"`
		LinkID          string `json:"link_id"`
		AppVersion      string `json:"app_version"`
		DocumentPolicy  string `json:"document_policy"`
		PatreonClientID string `json:"patreon_client_id"`
		DiscordClientID string `json:"discord_client_id"`
		DiscordEnabled  bool   `json:"discord_enabled"`
		PatreonEnabled  bool   `json:"patreon_enabled"`
		DefaultRoute    string `json:"default_route"`
		NewsEnabled     bool   `json:"news_enabled"`
		ForumsEnabled   bool   `json:"forums_enabled"`
		ContestsEnabled bool   `json:"contests_enabled"`
		WikiEnabled     bool   `json:"wiki_enabled"`
		StatsEnabled    bool   `json:"stats_enabled"`
		ServersEnabled  bool   `json:"servers_enabled"`
		ReportsEnabled  bool   `json:"reports_enabled"`
		ChatlogsEnabled bool   `json:"chatlogs_enabled"`
		DemosEnabled    bool   `json:"demos_enabled"`
	}

	return func(ctx *gin.Context) {
		conf := c.cu.Config()

		ctx.JSON(http.StatusOK, appInfo{
			SiteName:        conf.General.SiteName,
			AssetURL:        conf.General.AssetURL,
			LinkID:          conf.Discord.LinkID,
			AppVersion:      buildInfo.BuildVersion,
			DocumentPolicy:  "",
			PatreonClientID: conf.Patreon.ClientID,
			DiscordClientID: conf.Discord.AppID,
			DiscordEnabled:  conf.Discord.IntegrationsEnabled && conf.Discord.Enabled,
			PatreonEnabled:  conf.Patreon.IntegrationsEnabled && conf.Patreon.Enabled,
			DefaultRoute:    conf.General.DefaultRoute,
			NewsEnabled:     conf.General.NewsEnabled,
			ForumsEnabled:   conf.General.ForumsEnabled,
			ContestsEnabled: conf.General.ContestsEnabled,
			WikiEnabled:     conf.General.WikiEnabled,
			StatsEnabled:    conf.General.StatsEnabled,
			ServersEnabled:  conf.General.ServersEnabled,
			ReportsEnabled:  conf.General.ReportsEnabled,
			ChatlogsEnabled: conf.General.ChatlogsEnabled,
		})
	}
}
