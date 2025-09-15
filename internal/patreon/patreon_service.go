package patreon

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type patreonHandler struct {
	patreon PatreonUsecase
	config  *config.ConfigUsecase
}

func NewHandler(engine *gin.Engine, patreon PatreonUsecase, auth httphelper.Authenticator, config *config.ConfigUsecase) {
	handler := patreonHandler{
		patreon: patreon,
		config:  config,
	}

	engine.GET("/api/patreon/campaigns", handler.onAPIGetPatreonCampaigns())
	engine.GET("/patreon/oauth", handler.onOAuth())

	authGrp := engine.Group("/")
	{
		authed := authGrp.Use(auth.Middleware(permission.PUser))
		authed.GET("/api/patreon/login", handler.onLogin())
		authed.GET("/api/patreon/logout", handler.onLogout())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(permission.PModerator))
		mod.GET("/api/patreon/pledges", handler.onAPIGetPatreonPledges())
	}
}

func (h patreonHandler) onLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		if err := h.patreon.Forget(ctx, currentUser.GetSteamID()); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"url": h.patreon.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User removed their patreon credentials", slog.String("sid", sid.String()))
	}
}

func (h patreonHandler) onLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		ctx.JSON(http.StatusOK, gin.H{"url": h.patreon.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User tried to connect patreon", slog.String("sid", sid.String()))
	}
}

func (h patreonHandler) onOAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		grantCode, codeOK := ctx.GetQuery("code")
		if !codeOK {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, domain.ErrInvalidParameter, "code invalid."))

			return
		}

		state, stateOK := ctx.GetQuery("state")
		if !stateOK {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, domain.ErrInvalidParameter, "state invalid."))

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
