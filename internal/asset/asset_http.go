package asset

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type assetHandler struct {
	Assets
}

func NewAssetHandler(engine *gin.Engine, assets Assets) {
	// FIXME add auth
	handler := assetHandler{Assets: assets}
	engine.GET("/asset/:asset_id", handler.getAsset())
	// optGrp := engine.Group("/")
	// {
	//   opt := optGrp.Use(authenticator.Middleware(permission.Guest))
	//	 opt.GET("/asset/:asset_id", handler.getAsset())
	// }
}

func (h assetHandler) getAsset() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idFound := httphelper.GetUUIDParam(ctx, "asset_id")
		if !idFound {
			return
		}

		asset, errGet := h.Get(ctx, mediaID)
		if errGet != nil {
			if errors.Is(errGet, ErrOpenFile) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Asset with this asset_id does not exist: %s", mediaID))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}
		defer asset.Close()

		if asset.IsPrivate {
			user, _ := session.CurrentUserProfile(ctx)
			sid := user.GetSteamID()
			if !sid.Valid() || sid != asset.AuthorID && !user.HasPermission(permission.Moderator) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, permission.ErrDenied,
					"You do not have permission to access this asset."))

				return
			}
		}

		// TODO find out why the zstd reader does not read and close properly. Just caveman it and read,
		//  it all into memory for now.
		// header := map[string]string{
		// 	"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, asset.String()),
		// }
		decodedBody, errDecode := io.ReadAll(&asset)
		if errDecode != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDecode, httphelper.ErrInternal)))

			return
		}

		ctx.Writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, asset.String()))
		ctx.Data(http.StatusOK, asset.MimeType, decodedBody)
	}
}
