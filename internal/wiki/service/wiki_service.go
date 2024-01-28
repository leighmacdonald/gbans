package service

import (
	"errors"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"go.uber.org/zap"
)

type WikiHandler struct {
	wikiUsecase domain.WikiUsecase
	log         *zap.Logger
}

func NewWIkiHandler(logger *zap.Logger, engine *gin.Engine, wikiUsecase domain.WikiUsecase) {
	handler := &WikiHandler{
		wikiUsecase: wikiUsecase,
		log:         logger.Named("wiki"),
	}

	// optional
	engine.GET("/api/wiki/slug/*slug", handler.onAPIGetWikiSlug())

	// editor
	engine.POST("/api/wiki/slug", handler.onAPISaveWikiSlug())
}

func (w *WikiHandler) onAPIGetWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := http_helper.CurrentUserProfile(ctx)

		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		var page wiki.Page
		if errGetWikiSlug := w.wikiUsecase.GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, domain.ErrNoResult) {
				ctx.JSON(http.StatusOK, page)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if page.PermissionLevel > currentUser.PermissionLevel {
			http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func (w *WikiHandler) onAPISaveWikiSlug() gin.HandlerFunc {
	log := w.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req wiki.Page
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Slug == "" || req.BodyMD == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var page wiki.Page
		if errGetWikiSlug := w.wikiUsecase.GetWikiPageBySlug(ctx, req.Slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, domain.ErrNoResult) {
				page.CreatedOn = time.Now()
				page.Revision += 1
				page.Slug = req.Slug
			} else {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}
		} else {
			page = page.NewRevision()
		}

		page.PermissionLevel = req.PermissionLevel
		page.BodyMD = req.BodyMD

		if errSave := w.wikiUsecase.SaveWikiPage(ctx, &page); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}
