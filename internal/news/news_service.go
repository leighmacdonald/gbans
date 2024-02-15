package news

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type newsHandler struct {
	newsUsecase domain.NewsUsecase
	du          domain.DiscordUsecase
}

func NewNewsHandler(engine *gin.Engine, nu domain.NewsUsecase, du domain.DiscordUsecase, ath domain.AuthUsecase) {
	handler := newsHandler{newsUsecase: nu, du: du}

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

func (h newsHandler) onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.newsUsecase.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func (h newsHandler) onAPIPostNewsCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.NewsEntry
		if !httphelper.Bind(ctx, &req) {
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

func (h newsHandler) onAPIPostNewsUpdate() gin.HandlerFunc {
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

		if !httphelper.Bind(ctx, &entry) {
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

func (h newsHandler) onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.newsUsecase.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}
