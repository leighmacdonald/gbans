package wiki

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wikiHandler struct {
	Wiki
}

func NewWikiHandler(engine *gin.Engine, wiki Wiki, ath httphelper.Authenticator) {
	handler := &wikiHandler{wiki}

	// optional
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.Middleware(permission.Guest))
		opt.GET("/api/wiki/slug/:slug", handler.savePage())
	}

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.Middleware(permission.Editor))
		// TODO use PUT and slug param
		editor.PUT("/api/wiki/slug/:slug", handler.getPage())
	}
}

func (w *wikiHandler) savePage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slug := ctx.Param("slug")
		user, _ := session.CurrentUserProfile(ctx)
		page, err := w.Page(ctx, slug)
		if err != nil {
			switch {
			case errors.Is(err, database.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, errors.Join(err, httphelper.ErrNotFound)))
			case errors.Is(err, httphelper.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, errors.Join(err, httphelper.ErrPermissionDenied)))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))
			}

			return
		}

		if !user.HasPermission(page.PermissionLevel) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, errors.Join(err, httphelper.ErrPermissionDenied)))

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func (w *wikiHandler) getPage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Page
		if !httphelper.Bind(ctx, &req) {
			return
		}
		slugParam := ctx.Param("slug")
		page, err := w.Page(ctx, slugParam)
		if err != nil && errors.Is(err, ErrSlugUnknown) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		// if !user.HasPermission(page.PermissionLevel) {
		// 	httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, errors.Join(err, httphelper.ErrPermissionDenied)))

		// 	return
		// }

		page.Slug = req.Slug
		page.BodyMD = req.BodyMD
		page.PermissionLevel = req.PermissionLevel

		newPage, errSave := w.Save(ctx, page)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, newPage)
	}
}
