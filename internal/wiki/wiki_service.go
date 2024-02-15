package wiki

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wikiHandler struct {
	wikiUsecase domain.WikiUsecase
}

func NewWIkiHandler(engine *gin.Engine, wikiUsecase domain.WikiUsecase, ath domain.AuthUsecase) {
	handler := &wikiHandler{wikiUsecase: wikiUsecase}

	// optional
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.AuthMiddleware(domain.PGuest))
		opt.GET("/api/wiki/slug/*slug", handler.onAPIGetWikiSlug())
	}

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.AuthMiddleware(domain.PEditor))
		editor.POST("/api/wiki/slug", handler.onAPISaveWikiSlug())
	}
}

func (w *wikiHandler) onAPIGetWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		page, err := w.wikiUsecase.GetWikiPageBySlug(ctx, httphelper.CurrentUserProfile(ctx), ctx.Param("slug"))
		if err != nil {
			httphelper.ErrorHandled(ctx, err)

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

		page, err := w.wikiUsecase.SaveWikiPage(ctx, httphelper.CurrentUserProfile(ctx), req.Slug, req.BodyMD, req.PermissionLevel)
		if err != nil {
			httphelper.ErrorHandled(ctx, err)

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}
