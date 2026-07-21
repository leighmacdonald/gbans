package discordoauth

import (
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/person"
)

type discordOAuthHandler struct {
	DiscordOAuth

	config  *config.Configuration
	persons *person.Persons
}

func NewDiscordOAuthHandler(mux *http.ServeMux, config *config.Configuration,
	persons *person.Persons, discord DiscordOAuth,
) {
	handler := discordOAuthHandler{
		DiscordOAuth: discord,
		config:       config,
		persons:      persons,
	}

	mux.HandleFunc("GET /discord/oauth", handler.onOAuthDiscordCallback())
}

func (h discordOAuthHandler) onOAuthDiscordCallback() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
		if code == "" {
			slog.Error("Failed to get code from query")
			http.Redirect(res, req, link.Raw("/settings?section=connections"), http.StatusTemporaryRedirect)

			return
		}

		state := req.URL.Query().Get("state")
		if state == "" {
			slog.Error("Failed to get state from query")
			http.Redirect(res, req, link.Raw("/settings?section=connections"), http.StatusTemporaryRedirect)

			return
		}

		if err := h.HandleOAuthCode(req.Context(), code, state); err != nil {
			slog.Error("Failed to get access token", slog.String("error", err.Error()))
		}

		http.Redirect(res, req, link.Raw("/settings?section=connections"), http.StatusTemporaryRedirect)
	}
}
