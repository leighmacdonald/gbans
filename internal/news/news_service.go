package news

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type NewsHandler struct {
	newsUsecase domain.NewsUsecase
	du          domain.DiscordUsecase
	log         *zap.Logger
}

func NewNewsHandler(logger *zap.Logger, engine *gin.Engine, nu domain.NewsUsecase, du domain.DiscordUsecase, ath domain.AuthUsecase) {
	handler := NewsHandler{log: logger.Named("news"), newsUsecase: nu, du: du}

	engine.POST("/api/news_latest", handler.onAPIGetNewsLatest())

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.AuthMiddleware(domain.PUser))
		editor.POST("/api/news", handler.onAPIPostNewsCreate())
		editor.POST("/api/news/:news_id", handler.onAPIPostNewsUpdate())
		editor.POST("/api/news_all", handler.onAPIGetNewsAll())
	}
}

func (h NewsHandler) onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.newsUsecase.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func (h NewsHandler) onAPIPostNewsCreate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.NewsEntry
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if errSave := h.newsUsecase.SaveNewsArticle(ctx, &req); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, req)

		go h.du.SendPayload(domain.ChannelModLog, discord.NewNewsMessage(req.BodyMD, req.Title))
	}
}

func (h NewsHandler) onAPIPostNewsUpdate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newsID, errID := httphelper.GetIntParam(ctx, "news_id")
		if errID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var entry domain.NewsEntry
		if errGet := h.newsUsecase.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if !httphelper.Bind(ctx, log, &entry) {
			return
		}

		if errSave := h.newsUsecase.SaveNewsArticle(ctx, &entry); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		h.du.SendPayload(domain.ChannelModLog, discord.EditNewsMessages(entry.Title, entry.BodyMD))
	}
}

func (h NewsHandler) onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.newsUsecase.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}
