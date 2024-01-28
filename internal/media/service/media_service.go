package service

import (
	"bytes"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

func onAPISaveMedia() gin.HandlerFunc {
	MediaSafeMimeTypesImages := []string{
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req UserUploadedFile
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

		conf := env.Config()

		asset, errAsset := domain.NewAsset(content, conf.S3.BucketMedia, "")
		if errAsset != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetCreateFailed)

			return
		}

		if errPut := env.Assets().Put(ctx, conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetPut)

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := env.Store().SaveAsset(ctx, &asset); errSaveAsset != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetSave)

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !slices.Contains(MediaSafeMimeTypesImages, media.MimeType) {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidFormat)
			log.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := env.Store().SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save wiki media", zap.Error(errSave))

			if errors.Is(errSave, errs.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicateMediaName)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrSaveMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func onGetMediaByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var media domain.Media
		if errMedia := env.Store().GetMediaByID(ctx, mediaID, &media); errMedia != nil {
			if errors.Is(errs.DBErr(errMedia), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			}

			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}
