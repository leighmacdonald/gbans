package asset

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type mediaHandler struct {
	assets domain.AssetUsecase
	config domain.ConfigUsecase
}

func NewAssetHandler(engine *gin.Engine, config domain.ConfigUsecase, assets domain.AssetUsecase, auth domain.AuthUsecase) {
	handler := mediaHandler{config: config, assets: assets}

	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(auth.AuthMiddleware(domain.PGuest))
		opt.GET("/asset/:asset_id", handler.onGetByUUID())
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.POST("/api/asset", handler.onAPISaveMedia())
	}
}

func (h mediaHandler) onAPISaveMedia() gin.HandlerFunc {
	handlerName := log.HandlerName(1)

	return func(ctx *gin.Context) {
		var req domain.UserUploadedFile

		if !httphelper.Bind(ctx, &req) {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to bind request", handlerName)

			return
		}

		mediaFile, errOpen := req.File.Open()
		if errOpen != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to open form file", log.ErrAttr(errOpen), handlerName)

			return
		}

		if req.Name == "" {
			req.Name = req.File.Filename
		}

		media, errMedia := h.assets.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID, "media", req.Name, mediaFile)
		if errMedia != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errMedia)
			slog.Error("Failed to create new asset", log.ErrAttr(errMedia), handlerName)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h mediaHandler) onGetByUUID() gin.HandlerFunc {
	handlerName := log.HandlerName(1)

	return func(ctx *gin.Context) {
		mediaID, idErr := httphelper.GetUUIDParam(ctx, "asset_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Error("Got invalid asset_id", handlerName)

			return
		}

		asset, reader, errGet := h.assets.Get(ctx, mediaID)
		if errGet != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errGet)
			slog.Error("Failed to load asset", slog.String("asset_id", mediaID.String()), handlerName)

			return
		}

		if asset.IsPrivate {
			user := httphelper.CurrentUserProfile(ctx)
			if !user.SteamID.Valid() && (user.SteamID == asset.AuthorID || user.HasPermission(domain.PModerator)) {
				httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)
				slog.Warn("Tried to access private asset", slog.String("asset_id", mediaID.String()), handlerName)

				return
			}
		}

		header := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, asset.Name),
		}

		ctx.DataFromReader(http.StatusOK, asset.Size, asset.MimeType, reader, header)
	}
}
