package media

import (
	"encoding/base64"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
)

type mediaHandler struct {
	au  domain.AssetUsecase
	mu  domain.MediaUsecase
	cu  domain.ConfigUsecase
	log *zap.Logger
}

func NewMediaHandler(logger *zap.Logger, engine *gin.Engine, mu domain.MediaUsecase, cu domain.ConfigUsecase, au domain.AssetUsecase, ath domain.AuthUsecase) {
	handler := mediaHandler{
		mu:  mu,
		cu:  cu,
		au:  au,
		log: logger.Named("media"),
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
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.UserUploadedFile
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		media, errMedia := h.mu.Create(ctx, http_helper.CurrentUserProfile(ctx).SteamID, req.Name, req.Mime, content, nil)

		if errMedia != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, errMedia)
			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h mediaHandler) onGetMediaByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := http_helper.GetIntParam(ctx, "media_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var media domain.Media
		if err := http_helper.ErrorHandled(ctx, h.mu.GetMediaByID(ctx, mediaID, &media)); err != nil {
			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}
