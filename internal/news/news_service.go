package news

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type newsHandler struct {
	news    domain.NewsUsecase
	discord domain.DiscordUsecase
}

func NewNewsHandler(engine *gin.Engine, news domain.NewsUsecase, discord domain.DiscordUsecase, auth domain.AuthUsecase) {
	handler := newsHandler{news: news, discord: discord}

	engine.POST("/api/news_latest", handler.onAPIGetNewsLatest())

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(auth.AuthMiddleware(domain.PUser))
		editor.POST("/api/news", handler.onAPIPostNewsCreate())
		editor.POST("/api/news/:news_id", handler.onAPIPostNewsUpdate())
		editor.DELETE("/api/news/:news_id", handler.onAPIPostNewsDelete())
		editor.POST("/api/news_all", handler.onAPIGetNewsAll())
	}
}

func (h newsHandler) onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.news.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load news", log.ErrAttr(errGetNewsLatest))

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

type newsEditRequest struct {
	Title       string `json:"title"`
	BodyMD      string `json:"body_md"`
	IsPublished bool   `json:"is_published"`
}

func (h newsHandler) onAPIPostNewsCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req newsEditRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		entry := domain.NewsEntry{
			Title:       req.Title,
			BodyMD:      req.BodyMD,
			IsPublished: req.IsPublished,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := h.news.SaveNewsArticle(ctx, &entry); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save news article", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		go h.discord.SendPayload(domain.ChannelModLog, discord.NewNewsMessage(req.BodyMD, req.Title))
	}
}

func (h newsHandler) onAPIPostNewsUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, errID := httphelper.GetIntParam(ctx, "news_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get news_id", log.ErrAttr(errID))

			return
		}

		var entry domain.NewsEntry
		if errGet := h.news.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
				slog.Warn("Failed to get news by id. Not found.", log.ErrAttr(errGet))

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Warn("Failed to get news by id. Not found.", log.ErrAttr(errGet))

			return
		}

		var req newsEditRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		entry.Title = req.Title
		entry.BodyMD = req.BodyMD
		entry.IsPublished = req.IsPublished
		entry.UpdatedOn = time.Now()

		if errSave := h.news.SaveNewsArticle(ctx, &entry); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save news article", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		h.discord.SendPayload(domain.ChannelModLog, discord.EditNewsMessages(entry.Title, entry.BodyMD))
	}
}

func (h newsHandler) onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.news.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get latest news", log.ErrAttr(errGetNewsLatest))

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func (h newsHandler) onAPIPostNewsDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, errID := httphelper.GetIntParam(ctx, "news_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get news_id", log.ErrAttr(errID))

			return
		}

		var entry domain.NewsEntry
		if errGet := h.news.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get news by ID", log.ErrAttr(errGet))

			return
		}

		if err := h.news.DropNewsArticle(ctx, newsID); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete news entry", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
