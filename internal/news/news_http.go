package news

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type newsHandler struct {
	News
}

func NewNewsHandler(engine *gin.Engine, news News, auth httphelper.Authenticator) {
	handler := newsHandler{News: news}

	engine.GET("/api/news_latest", handler.onAPIGetNewsLatest())

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(auth.Middleware(permission.Editor))
		editor.POST("/api/news", handler.onAPIPostNewsCreate())
		editor.POST("/api/news/:news_id", handler.onAPIPostNewsUpdate())
		editor.DELETE("/api/news/:news_id", handler.onAPIPostNewsDelete())
		editor.GET("/api/news_all", handler.onAPIGetNewsAll())
	}
}

func (h newsHandler) onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetNewsLatest, httphelper.ErrInternal)))

			return
		}

		if newsLatest == nil {
			newsLatest = []Article{}
		}

		ctx.PureJSON(http.StatusOK, newsLatest)
	}
}

type EditRequest struct {
	Title       string `json:"title"`
	BodyMD      string `json:"body_md"`
	IsPublished bool   `json:"is_published"`
}

func (h newsHandler) onAPIPostNewsCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req EditRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		entry := Article{
			Title:       req.Title,
			BodyMD:      req.BodyMD,
			IsPublished: req.IsPublished,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := h.Save(ctx, &entry); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		// h.notifications.Enqueue(ctx, notification.NewDiscordNotification(
		// 	discord.ChannelModLog,
		// 	message.NewNewsMessage(req.BodyMD, req.Title)))
	}
}

func (h newsHandler) onAPIPostNewsUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, idFound := httphelper.GetIntParam(ctx, "news_id")
		if !idFound {
			return
		}

		var entry Article
		if errGet := h.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		var req EditRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		entry.Title = req.Title
		entry.BodyMD = req.BodyMD
		entry.IsPublished = req.IsPublished
		entry.UpdatedOn = time.Now()

		if errSave := h.Save(ctx, &entry); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, entry)

		// h.notifications.Enqueue(ctx, notification.NewDiscordNotification(
		// 	discord.ChannelModLog,
		// 	message.EditNewsMessages(entry.Title, entry.BodyMD)))
	}
}

func (h newsHandler) onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := h.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetNewsLatest, httphelper.ErrInternal)))

			return
		}

		if newsLatest == nil {
			newsLatest = []Article{}
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

		var entry Article
		if errGet := h.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errGet, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		if err := h.DropNewsArticle(ctx, newsID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
