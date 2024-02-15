package media

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type mediaHandler struct {
	au domain.AssetUsecase
	mu domain.MediaUsecase
	cu domain.ConfigUsecase
}

func NewMediaHandler(engine *gin.Engine, mu domain.MediaUsecase, cu domain.ConfigUsecase, au domain.AssetUsecase, ath domain.AuthUsecase) {
	handler := mediaHandler{
		mu: mu,
		cu: cu,
		au: au,
	}

	engine.GET("/media/:media_id", handler.onGetMediaByID())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/media", handler.onAPISaveMedia())
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

		media, errMedia := h.mu.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID, req.Name, req.Mime, content, nil)

		if errMedia != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h mediaHandler) onGetMediaByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := httphelper.GetIntParam(ctx, "media_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var media domain.Media
		if err := httphelper.ErrorHandledWithReturn(ctx, h.mu.GetMediaByID(ctx, mediaID, &media)); err != nil {
			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}
