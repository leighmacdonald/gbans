package patreon

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type patreonHandler struct {
	patreon domain.PatreonUsecase
	config  domain.ConfigUsecase
}

func NewPatreonHandler(engine *gin.Engine, patreon domain.PatreonUsecase, auth domain.AuthUsecase, config domain.ConfigUsecase) {
	handler := patreonHandler{
		patreon: patreon,
		config:  config,
	}

	engine.GET("/api/patreon/campaigns", handler.onAPIGetPatreonCampaigns())
	engine.GET("/patreon/oauth", handler.onOAuth())

	authGrp := engine.Group("/")
	{
		authed := authGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.GET("/api/patreon/login", handler.onLogin())
		authed.GET("/api/patreon/logout", handler.onLogout())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
		mod.GET("/api/patreon/pledges", handler.onAPIGetPatreonPledges())
	}
}

func (h patreonHandler) onLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		if err := h.patreon.Forget(ctx, currentUser.SteamID); err != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, nil)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"url": h.patreon.CreateOAuthRedirect(currentUser.SteamID)})
		slog.Debug("User removed their patreon credentials", slog.String("sid", currentUser.SteamID.String()))
	}
}

func (h patreonHandler) onLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		ctx.JSON(http.StatusOK, gin.H{"url": h.patreon.CreateOAuthRedirect(currentUser.SteamID)})
		slog.Debug("User tried to connect patreon", slog.String("sid", currentUser.SteamID.String()))
	}
}

func (h patreonHandler) onOAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		grantCode, codeOK := ctx.GetQuery("code")
		if !codeOK {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, nil)

			return
		}

		state, stateOK := ctx.GetQuery("state")
		if !stateOK {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if err := h.patreon.OnOauthLogin(ctx, state, grantCode); err != nil {
			slog.Error("Failed to handle oauth login", log.ErrAttr(err))
		} else {
			slog.Debug("Successfully authenticated user over patreon")
		}

		conf := h.config.Config()

		ctx.Redirect(http.StatusPermanentRedirect, conf.ExtURLRaw("/patreon"))
	}
}

func (h patreonHandler) onAPIGetPatreonCampaigns() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, h.patreon.Campaign())
	}
}

func (h patreonHandler) onAPIGetPatreonPledges() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{})
	}
}
