package asset

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type assetHandler struct {
	Assets
}

func NewAssetHandler(mux *http.ServeMux, assets Assets) {
	handler := assetHandler{Assets: assets}
	mux.HandleFunc("GET /asset/{asset_id}", handler.getAsset())
}

func (h assetHandler) getAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaID, idFound := httphelper.GetUUIDParam(r, "asset_id")
		if !idFound {
			return
		}

		asset, errGet := h.Get(r.Context(), mediaID)
		if errGet != nil {
			if errors.Is(errGet, ErrOpenFile) {
				httphelper.SetError(w, r, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Asset with this asset_id does not exist: %s", mediaID))

				return
			}

			httphelper.SetError(w, r, httphelper.NewAPIError(http.StatusBadRequest, errGet))

			return
		}
		defer func(asset *Asset) {
			if err := asset.Close(); err != nil {
				slog.Error("Fauled to close asset")
			}
		}(&asset)

		if asset.IsPrivate {
			user, errProfile := session.CurrentUserProfile(r.Context())
			if errProfile != nil {
				slog.Error("Failed to get user session for private asset", slog.String("error", errProfile.Error()))
			}

			sid := user.GetSteamID()
			if !sid.Valid() || sid != asset.AuthorID && !user.HasPermission(permission.Moderator) {
				httphelper.SetError(w, r, httphelper.NewAPIErrorf(http.StatusForbidden, permission.ErrDenied,
					"You do not have permission to access this asset."))

				return
			}
		}

		decodedBody, errDecode := io.ReadAll(&asset)
		if errDecode != nil {
			httphelper.SetError(w, r, httphelper.NewAPIError(http.StatusInternalServerError, errDecode))

			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, asset.String()))
		w.Header().Set("Content-Type", asset.MimeType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(decodedBody)
	}
}
