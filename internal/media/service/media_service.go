package service

import (
	"encoding/base64"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
)

type MediaHandler struct {
	au  domain.AssetUsecase
	mu  domain.MediaUsecase
	cu  domain.ConfigUsecase
	log *zap.Logger
}

func NewMediaHandler(logger *zap.Logger, engine *gin.Engine, mu domain.MediaUsecase, cu domain.ConfigUsecase, au domain.AssetUsecase) {
	handler := MediaHandler{
		mu:  mu,
		cu:  cu,
		au:  au,
		log: logger.Named("media"),
	}

	engine.GET("/media/:media_id", handler.onGetMediaByID())

	// authed
	engine.POST("/api/media", handler.onAPISaveMedia())
}

func (h MediaHandler) onAPISaveMedia() gin.HandlerFunc {
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

		media, errMedia := domain.NewMedia(http_helper.CurrentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		if http_helper.ErrorHandled(ctx, h.mu.SaveMedia(ctx, &media)) {
			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (h MediaHandler) onGetMediaByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := http_helper.GetIntParam(ctx, "media_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var media domain.Media
		if http_helper.ErrorHandled(ctx, h.mu.GetMediaByID(ctx, mediaID, &media)) {
			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}
