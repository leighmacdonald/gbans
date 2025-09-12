package asset

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type mediaHandler struct {
	assets AssetUsecase
}

func NewHandler(engine *gin.Engine, assets AssetUsecase, authenticator httphelper.Authenticator) {
	handler := mediaHandler{assets: assets}

	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(authenticator.Middleware(permission.PGuest))
		opt.GET("/asset/:asset_id", handler.onGetByUUID())
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.PUser))
		authed.POST("/api/asset", handler.onAPISaveMedia())
	}
}

func (h mediaHandler) onAPISaveMedia() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req UserUploadedFile

		if err := ctx.Bind(&req); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, err))

			return
		}

		mediaFile, errOpen := req.File.Open()
		if errOpen != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errOpen, httphelper.ErrInternal)))

			return
		}

		if req.Name == "" {
			req.Name = req.File.Filename
		}
		user, _ := session.CurrentUserProfile(ctx)
		media, errMedia := h.assets.Create(ctx, user.GetSteamID(), "media", req.Name, mediaFile)
		if errMedia != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMedia, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h mediaHandler) onGetByUUID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idFound := httphelper.GetUUIDParam(ctx, "asset_id")
		if !idFound {
			return
		}

		asset, reader, errGet := h.assets.Get(ctx, mediaID)
		if errGet != nil {
			if errors.Is(errGet, domain.ErrOpenFile) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Asset with this asset_id does not exist: %s", mediaID))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		if asset.IsPrivate {
			user, _ := session.CurrentUserProfile(ctx)
			sid := user.GetSteamID()
			if !sid.Valid() && (sid == asset.AuthorID || user.HasPermission(permission.PModerator)) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
					"You do not have permission to access this asset."))

				return
			}
		}

		header := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, asset.Name),
		}

		ctx.DataFromReader(http.StatusOK, asset.Size, asset.MimeType, reader, header)
	}
}
