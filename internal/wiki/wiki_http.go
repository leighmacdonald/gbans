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
	wiki Wiki
}

func NewWikiHandler(engine *gin.Engine, wiki Wiki, ath httphelper.Authenticator) {
	handler := &wikiHandler{wiki: wiki}

	// optional
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.Middleware(permission.Guest))
		opt.GET("/api/wiki/slug/*slug", handler.onAPIGetWikiSlug())
	}

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.Middleware(permission.Editor))
		// TODO use PUT and slug param
		editor.POST("/api/wiki/slug", handler.onAPISaveWikiSlug())
	}
}

func (w *wikiHandler) onAPIGetWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		page, err := w.wiki.BySlug(ctx, user, ctx.Param("slug"))
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

		ctx.JSON(http.StatusOK, page)
	}
}

func (w *wikiHandler) onAPISaveWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Page
		if !httphelper.Bind(ctx, &req) {
			return
		}
		user, _ := session.CurrentUserProfile(ctx)
		page, err := w.wiki.Save(ctx, user, req.Slug, req.BodyMD, req.PermissionLevel)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}
