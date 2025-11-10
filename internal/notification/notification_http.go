package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type messagesRequest struct {
	MessageIDs []int `json:"message_ids"` //nolint:tagliatelle
}

type notificationHandler struct {
	*Notifications
}

func NewNotificationHandler(engine *gin.Engine, auth httphelper.Authenticator, notifications *Notifications) {
	handler := notificationHandler{
		Notifications: notifications,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.Middleware(permission.User))
		authed.GET("/api/notifications", handler.onNotifications())
		authed.POST("/api/notifications/all", handler.onMarkAllRead())
		authed.POST("/api/notifications", handler.onMarkRead())
		authed.DELETE("/api/notifications/all", handler.onDeleteAll())
		authed.DELETE("/api/notifications", handler.onDelete())
	}
}

func (h notificationHandler) onMarkAllRead() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		if err := h.MarkAllRead(ctx, user.GetSteamID()); err != nil && !errors.Is(err, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onMarkRead() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request messagesRequest
		if !httphelper.Bind(ctx, &request) {
			return
		}

		if len(request.MessageIDs) == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"No message_ids value provided"))

			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		if err := h.MarkMessagesRead(ctx, user.GetSteamID(), request.MessageIDs); err != nil && !errors.Is(err, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onDeleteAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		if err := h.DeleteAll(ctx, user.GetSteamID()); err != nil && !errors.Is(err, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request messagesRequest
		if !httphelper.Bind(ctx, &request) {
			return
		}

		if len(request.MessageIDs) == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"message_ids cannot be empty."))

			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		if err := h.DeleteMessages(ctx, user.GetSteamID(), request.MessageIDs); err != nil && !errors.Is(err, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onNotifications() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		notifications, err := h.GetPersonNotifications(ctx, user.GetSteamID())
		if err != nil {
			if errors.Is(err, database.ErrNoResult) {
				ctx.JSON(http.StatusOK, []UserNotification{})

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, notifications)
	}
}
