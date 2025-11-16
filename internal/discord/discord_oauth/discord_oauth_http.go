package discordoauth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
)

type discordOAuthHandler struct {
	DiscordOAuth

	config  *config.Configuration
	persons *person.Persons
}

// NewDiscordOAuthHandler provides handlers for authentication with discord connect.
func NewDiscordOAuthHandler(engine *gin.Engine, auth httphelper.Authenticator, config *config.Configuration,
	persons *person.Persons, discord DiscordOAuth,
) {
	handler := discordOAuthHandler{
		DiscordOAuth: discord,
		config:       config,
		persons:      persons,
	}

	engine.GET("/discord/oauth", handler.onOAuthDiscordCallback())

	authGrp := engine.Group("/")
	{
		// authed
		authed := authGrp.Use(auth.Middleware(permission.User))
		authed.GET("/api/discord/login", handler.onLogin())
		authed.GET("/api/discord/logout", handler.onLogout())
		authed.GET("/api/discord/user", handler.onGetDiscordUser())
	}
}

func (h discordOAuthHandler) onLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)
		sid := currentUser.GetSteamID()

		loginURL, errURL := h.CreateStatefulLoginURL(sid)
		if errURL != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, errors.Join(errURL, httphelper.ErrBadRequest),
				"Could not construct oauth login URL"))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"url": loginURL})
		slog.Debug("User tried to connect discord", slog.String("sid", sid.String()))
	}
}

func (h discordOAuthHandler) onOAuthDiscordCallback() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		code := ctx.Query("code")
		if code == "" {
			slog.Error("Failed to get code from query")
			ctx.Redirect(http.StatusTemporaryRedirect, link.Raw("/settings?section=connections"))

			return
		}

		state := ctx.Query("state")
		if state == "" {
			slog.Error("Failed to get state from query")
			ctx.Redirect(http.StatusTemporaryRedirect, link.Raw("/settings?section=connections"))

			return
		}

		if err := h.HandleOAuthCode(ctx, code, state); err != nil {
			slog.Error("Failed to get access token", slog.String("error", err.Error()))
		}

		ctx.Redirect(http.StatusTemporaryRedirect, link.Raw("/settings?section=connections"))
	}
}

func (h discordOAuthHandler) onGetDiscordUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)

		discord, errUser := h.GetUserDetail(ctx, user.GetSteamID())
		if errUser != nil {
			if errors.Is(errUser, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, httphelper.ErrInternal,
				"Failed to fetch discord user details"))

			return
		}

		ctx.JSON(http.StatusOK, discord)
	}
}

func (h discordOAuthHandler) onLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)

		errUser := h.Logout(ctx, user.GetSteamID())
		if errUser != nil {
			if errors.Is(errUser, database.ErrNoResult) {
				ctx.JSON(http.StatusOK, gin.H{})

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, httphelper.ErrInternal,
				"Could not perform discord logout."))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
