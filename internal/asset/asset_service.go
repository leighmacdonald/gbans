package asset

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type mediaHandler struct {
	assets domain.AssetUsecase
	config domain.ConfigUsecase
}

func NewHandler(engine *gin.Engine, config domain.ConfigUsecase, assets domain.AssetUsecase, auth domain.AuthUsecase) {
	handler := mediaHandler{config: config, assets: assets}

	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(auth.Middleware(domain.PGuest))
		opt.GET("/asset/:asset_id", handler.onGetByUUID())
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.Middleware(domain.PUser))
		authed.POST("/api/asset", handler.onAPISaveMedia())
	}
}

func (h mediaHandler) onAPISaveMedia() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.UserUploadedFile

		if !httphelper.Bind(ctx, &req) {
			return
		}

		mediaFile, errOpen := req.File.Open()
		if errOpen != nil {
			httphelper.SetAPIError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOpen))

			return
		}

		if req.Name == "" {
			req.Name = req.File.Filename
		}

		media, errMedia := h.assets.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID, "media", req.Name, mediaFile)
		if errMedia != nil {
			httphelper.SetAPIError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errMedia))

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
			if errors.Is(errGet, domain.ErrNotFound) {
				httphelper.SetAPIError(ctx, httphelper.NewAPIError(http.StatusNotFound, domain.ErrNotFound))

				return
			}

			httphelper.SetAPIError(ctx, httphelper.NewAPIError(http.StatusBadRequest, errGet))

			return
		}

		if asset.IsPrivate {
			user := httphelper.CurrentUserProfile(ctx)
			if !user.SteamID.Valid() && (user.SteamID == asset.AuthorID || user.HasPermission(domain.PModerator)) {
				httphelper.SetAPIError(ctx, httphelper.NewAPIError(http.StatusForbidden, domain.ErrPermissionDenied))

				return
			}
		}

		header := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, asset.Name),
		}

		ctx.DataFromReader(http.StatusOK, asset.Size, asset.MimeType, reader, header)
	}
}
