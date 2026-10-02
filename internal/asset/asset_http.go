package asset

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
)

type assetHandler struct {
	Assets

	roleAuth *rpc.RoleAuth
}

func NewAssetHandler(mux *http.ServeMux, assets Assets, roleAuth *rpc.RoleAuth) {
	handler := assetHandler{Assets: assets, roleAuth: roleAuth}
	mux.HandleFunc("GET /asset/{asset_id}", handler.getAsset())
}

func (h assetHandler) getAsset() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		mediaID, idFound := httphelper.GetUUIDParam(req, "asset_id")
		if !idFound {
			return
		}

		assetValue, errGet := h.Get(req.Context(), mediaID)
		if errGet != nil {
			if errors.Is(errGet, ErrOpenFile) {
				httphelper.SetError(res, req, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Asset with this asset_id does not exist: %s", mediaID))

				return
			}

			httphelper.SetError(res, req, httphelper.NewAPIError(http.StatusBadRequest, errGet))

			return
		}
		defer func(a *Asset) {
			if err := a.Close(); err != nil {
				slog.Error("Fauled to close asset")
			}
		}(&assetValue)

		if assetValue.IsPrivate {
			user, errProfile := session.CurrentUserProfile(req.Context())
			if errProfile != nil {
				slog.Error("Failed to get user session for private asset", slog.String("error", errProfile.Error()))
			}

			sid := user.GetSteamID()
			if !sid.Valid() || sid != assetValue.AuthorID && !h.roleAuth.HasPermissionForSteamID(req.Context(), sid, rolesv1.Permission_PERMISSION_BAN_READ) {
				httphelper.SetError(res, req, httphelper.NewAPIErrorf(http.StatusForbidden, rpc.ErrPermission,
					"You do not have permission to access this asset."))

				return
			}
		}

		decodedBody, errDecode := io.ReadAll(&assetValue)
		if errDecode != nil {
			httphelper.SetError(res, req, httphelper.NewAPIError(http.StatusInternalServerError, errDecode))

			return
		}

		res.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, assetValue.String()))
		res.Header().Set("Content-Type", assetValue.MimeType)
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write(decodedBody) //nolint:gosec
	}
}
