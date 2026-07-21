package patreon

import (
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type patreonHandler struct {
	Patreon

	config Config
}

// func NewPatreonHandler(mux *http.ServeMux, patreon Patreon, config Config) {
//	handler := patreonHandler{
//		Patreon: patreon,
//		config:  config,
//	}
//
//	mux.HandleFunc("GET /api/patreon/campaigns", handler.onAPIGetPatreonCampaigns())
//	mux.HandleFunc("GET /patreon/oauth", handler.onOAuth())
//	mux.HandleFunc("GET /api/patreon/login", handler.onLogin())
//	mux.HandleFunc("GET /api/patreon/logout", handler.onLogout())
//	mux.HandleFunc("GET /api/patreon/pledges", handler.onAPIGetPatreonPledges())
// }

func (h *patreonHandler) onLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := session.CurrentUserProfile(r.Context())

		if err := h.Forget(r.Context(), currentUser.GetSteamID()); err != nil {
			httphelper.SetError(w, r, httphelper.NewAPIError(http.StatusBadRequest, err))

			return
		}

		httphelper.RespondJSON(w, http.StatusOK, map[string]string{"url": h.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User removed their patreon credentials", slog.String("sid", sid.String()))
	}
}

func (h *patreonHandler) onLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := session.CurrentUserProfile(r.Context())

		httphelper.RespondJSON(w, http.StatusOK, map[string]string{"url": h.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User tried to connect patreon", slog.String("sid", sid.String()))
	}
}

func (h *patreonHandler) onOAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		grantCode := r.URL.Query().Get("code")
		if grantCode == "" {
			httphelper.SetError(w, r, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrInvalidParameter, "code invalid."))

			return
		}

		state := r.URL.Query().Get("state")
		if state == "" {
			httphelper.SetError(w, r, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrInvalidParameter, "state invalid."))

			return
		}

		if err := h.OnOauthLogin(r.Context(), state, grantCode); err != nil {
			slog.Error("Failed to handle oauth login", slog.String("error", err.Error()))
		} else {
			slog.Debug("Successfully authenticated user over patreon")
		}

		http.Redirect(w, r, link.Raw("/patreon"), http.StatusPermanentRedirect)
	}
}

func (h *patreonHandler) onAPIGetPatreonCampaigns() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httphelper.RespondJSON(w, http.StatusOK, h.Campaign())
	}
}

func (h *patreonHandler) onAPIGetPatreonPledges() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httphelper.RespondJSON(w, http.StatusOK, map[string]any{})
	}
}
