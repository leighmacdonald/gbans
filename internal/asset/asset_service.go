package asset

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type mediaHandler struct {
	au domain.AssetUsecase
	cu domain.ConfigUsecase
}

func NewAssetHandler(engine *gin.Engine, cu domain.ConfigUsecase, au domain.AssetUsecase, ath domain.AuthUsecase) {
	handler := mediaHandler{
		cu: cu,
		au: au,
	}
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.AuthMiddleware(domain.PGuest))
		opt.GET("/asset/:asset_id", handler.onGetByUUID())
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/asset", handler.onAPISaveMedia())
	}
}

func (h mediaHandler) onAPISaveMedia() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.UserUploadedFile
		if !httphelper.Bind(ctx, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		reader := bytes.NewReader(content)

		media, errMedia := h.au.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID, "media", req.Name, reader)
		if errMedia != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h mediaHandler) onGetByUUID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := httphelper.GetUUIDParam(ctx, "asset_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		asset, reader, errGet := h.au.Get(ctx, mediaID)
		if errGet != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errGet)

			return
		}

		if asset.IsPrivate {
			user := httphelper.CurrentUserProfile(ctx)
			if !user.SteamID.Valid() && (user.SteamID == asset.AuthorID || user.HasPermission(domain.PModerator)) {
				httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

				return
			}
		}

		header := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, asset.Name),
		}

		ctx.DataFromReader(http.StatusOK, asset.Size, asset.MimeType, reader, header)
	}
}
