package asset

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/zstd"
)

type mediaHandler struct {
	Assets
}

func NewAssetHandler(engine *gin.Engine, authenticator httphelper.Authenticator, assets Assets) {
	handler := mediaHandler{Assets: assets}

	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(authenticator.Middleware(permission.Guest))
		opt.GET("/asset/:asset_id", handler.getAsset())
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))
		authed.POST("/api/asset", handler.saveAsset())
	}
}

func (h mediaHandler) saveAsset() gin.HandlerFunc {
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
		asset, errAsset := h.Create(ctx, user.GetSteamID(), "media", req.Name, mediaFile, false)
		if errAsset != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errAsset, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, asset)
	}
}

func (h mediaHandler) getAsset() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idFound := httphelper.GetUUIDParam(ctx, "asset_id")
		if !idFound {
			return
		}

		asset, reader, errGet := h.Get(ctx, mediaID)
		if errGet != nil {
			if errors.Is(errGet, ErrOpenFile) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Asset with this asset_id does not exist: %s", mediaID))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		if asset.IsPrivate {
			user, _ := session.CurrentUserProfile(ctx)
			sid := user.GetSteamID()
			if !sid.Valid() || sid != asset.AuthorID && !user.HasPermission(permission.Moderator) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, permission.ErrDenied,
					"You do not have permission to access this asset."))

				return
			}
		}

		filename := asset.Name
		// Decompress it before sending out if its a zstd file
		if strings.HasSuffix(filename, zstd.Extension) {
			filename = strings.TrimSuffix(asset.Name, zstd.Extension)

			var decompressed bytes.Buffer
			if errDecompress := zstd.Decompress(reader, bufio.NewWriter(&decompressed)); errDecompress != nil {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDecompress, httphelper.ErrInternal)))

				return
			}

			reader = bytes.NewReader(decompressed.Bytes())
		}

		header := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, filename),
		}

		ctx.DataFromReader(http.StatusOK, asset.Size, asset.MimeType, reader, header)
	}
}
