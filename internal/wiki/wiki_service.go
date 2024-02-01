package wiki

import (
	"errors"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type wikiHandler struct {
	wikiUsecase domain.WikiUsecase
	log         *zap.Logger
}

func NewWIkiHandler(logger *zap.Logger, engine *gin.Engine, wikiUsecase domain.WikiUsecase, ath domain.AuthUsecase) {
	handler := &wikiHandler{
		wikiUsecase: wikiUsecase,
		log:         logger.Named("wiki"),
	}

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
		currentUser := httphelper.CurrentUserProfile(ctx)

		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		page, errGetWikiSlug := w.wikiUsecase.GetWikiPageBySlug(ctx, slug)
		if errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, domain.ErrNoResult) {
				ctx.JSON(http.StatusOK, page)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if page.PermissionLevel > currentUser.PermissionLevel {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func (w *wikiHandler) onAPISaveWikiSlug() gin.HandlerFunc {
	log := w.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.WikiPage
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if req.Slug == "" || req.BodyMD == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		page, errGetWikiSlug := w.wikiUsecase.GetWikiPageBySlug(ctx, req.Slug)
		if errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, domain.ErrNoResult) {
				page.CreatedOn = time.Now()
				page.Revision += 1
				page.Slug = req.Slug
			} else {
				httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}
		} else {
			page = page.NewRevision()
		}

		page.PermissionLevel = req.PermissionLevel
		page.BodyMD = req.BodyMD

		if errSave := w.wikiUsecase.SaveWikiPage(ctx, &page); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}
