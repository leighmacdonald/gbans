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

func (h *patreonHandler) onLogout() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		currentUser, _ := session.CurrentUserProfile(req.Context())

		if err := h.Forget(req.Context(), currentUser.GetSteamID()); err != nil {
			httphelper.SetError(res, req, httphelper.NewAPIError(http.StatusBadRequest, err))

			return
		}

		httphelper.RespondJSON(res, http.StatusOK, map[string]string{"url": h.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User removed their patreon credentials", slog.String("sid", sid.String()))
	}
}

func (h *patreonHandler) onLogin() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		currentUser, _ := session.CurrentUserProfile(req.Context())

		httphelper.RespondJSON(res, http.StatusOK, map[string]string{"url": h.CreateOAuthRedirect(currentUser.GetSteamID())})
		sid := currentUser.GetSteamID()
		slog.Debug("User tried to connect patreon", slog.String("sid", sid.String()))
	}
}

func (h *patreonHandler) onOAuth() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		grantCode := req.URL.Query().Get("code")
		if grantCode == "" {
			httphelper.SetError(res, req, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrInvalidParameter, "code invalid."))

			return
		}

		state := req.URL.Query().Get("state")
		if state == "" {
			httphelper.SetError(res, req, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrInvalidParameter, "state invalid."))

			return
		}

		if err := h.OnOauthLogin(req.Context(), state, grantCode); err != nil {
			slog.Error("Failed to handle oauth login", slog.String("error", err.Error()))
		} else {
			slog.Debug("Successfully authenticated user over patreon")
		}

		http.Redirect(res, req, link.Raw("/patreon"), http.StatusPermanentRedirect)
	}
}

func (h *patreonHandler) onAPIGetPatreonCampaigns() http.HandlerFunc {
	return func(res http.ResponseWriter, _ *http.Request) {
		httphelper.RespondJSON(res, http.StatusOK, h.Campaign())
	}
}

func (h *patreonHandler) onAPIGetPatreonPledges() http.HandlerFunc {
	return func(res http.ResponseWriter, _ *http.Request) {
		httphelper.RespondJSON(res, http.StatusOK, map[string]any{})
	}
}
