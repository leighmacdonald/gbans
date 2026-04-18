package discordoauth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/person"
)

type discordOAuthHandler struct {
	DiscordOAuth

	config  *config.Configuration
	persons *person.Persons
}

// NewDiscordOAuthHandler provides handlers for authentication with discord connect.
func NewDiscordOAuthHandler(engine *gin.Engine, config *config.Configuration,
	persons *person.Persons, discord DiscordOAuth,
) {
	handler := discordOAuthHandler{
		DiscordOAuth: discord,
		config:       config,
		persons:      persons,
	}

	engine.GET("/discord/oauth", handler.onOAuthDiscordCallback())
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
