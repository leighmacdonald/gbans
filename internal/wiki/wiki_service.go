package wiki

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wikiHandler struct {
	wiki domain.WikiUsecase
}

func NewHandler(engine *gin.Engine, wiki domain.WikiUsecase, ath domain.AuthUsecase) {
	handler := &wikiHandler{wiki: wiki}

	// optional
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.Middleware(domain.PGuest))
		opt.GET("/api/wiki/slug/*slug", handler.onAPIGetWikiSlug())
	}

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.Middleware(domain.PEditor))
		// TODO use PUT and slug param
		editor.POST("/api/wiki/slug", handler.onAPISaveWikiSlug())
	}
}

func (w *wikiHandler) onAPIGetWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		page, err := w.wiki.GetWikiPageBySlug(ctx, httphelper.CurrentUserProfile(ctx), ctx.Param("slug"))
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, errors.Join(err, domain.ErrNotFound)))
			case errors.Is(err, domain.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, errors.Join(err, domain.ErrPermissionDenied)))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func (w *wikiHandler) onAPISaveWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.WikiPage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		page, err := w.wiki.SaveWikiPage(ctx, httphelper.CurrentUserProfile(ctx), req.Slug, req.BodyMD, req.PermissionLevel)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}
