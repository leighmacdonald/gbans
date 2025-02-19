package news

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type newsHandler struct {
	news          domain.NewsUsecase
	notifications domain.NotificationUsecase
}

func NewHandler(engine *gin.Engine, news domain.NewsUsecase, notifications domain.NotificationUsecase, auth domain.AuthUsecase) {
	handler := newsHandler{news: news, notifications: notifications}

	engine.POST("/api/news_latest", handler.onAPIGetNewsLatest())

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(auth.Middleware(domain.PEditor))
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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetNewsLatest))

			return
		}

		if newsLatest == nil {
			newsLatest = []domain.NewsEntry{}
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

		if errSave := h.news.Save(ctx, &entry); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		h.notifications.Enqueue(ctx, domain.NewDiscordNotification(
			domain.ChannelModLog,
			discord.NewNewsMessage(req.BodyMD, req.Title)))
	}
}

func (h newsHandler) onAPIPostNewsUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, idFound := httphelper.GetIntParam(ctx, "news_id")
		if !idFound {
			return
		}

		var entry domain.NewsEntry
		if errGet := h.news.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

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

		if errSave := h.news.Save(ctx, &entry); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		h.notifications.Enqueue(ctx, domain.NewDiscordNotification(
			domain.ChannelModLog,
			discord.EditNewsMessages(entry.Title, entry.BodyMD)))
	}
}

func (h newsHandler) onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.news.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetNewsLatest))

			return
		}

		if newsLatest == nil {
			newsLatest = []domain.NewsEntry{}
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func (h newsHandler) onAPIPostNewsDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, idFound := httphelper.GetIntParam(ctx, "news_id")
		if !idFound {
			return
		}

		var entry domain.NewsEntry
		if errGet := h.news.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

			return
		}

		if err := h.news.DropNewsArticle(ctx, newsID); err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
