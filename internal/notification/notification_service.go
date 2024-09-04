package notification

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type messagesRequest struct {
	MessageIDs []int `json:"message_ids"` //nolint:tagliatelle
}

type notificationHandler struct {
	notifications domain.NotificationUsecase
}

func NewNotificationHandler(engine *gin.Engine, notifications domain.NotificationUsecase, auth domain.AuthUsecase) {
	handler := notificationHandler{
		notifications: notifications,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.GET("/api/notifications", handler.onNotifications())
		authed.POST("/api/notifications/all", handler.onMarkAllRead())
		authed.POST("/api/notifications", handler.onMarkRead())
		authed.DELETE("/api/notifications/all", handler.onDeleteAll())
		authed.DELETE("/api/notifications", handler.onDelete())
	}
}

func (h notificationHandler) onMarkAllRead() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := h.notifications.MarkAllRead(ctx, httphelper.CurrentUserProfile(ctx).SteamID); err != nil && !errors.Is(err, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to mark all notifications read", log.ErrAttr(err))

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
			httphelper.HandleErrs(ctx, domain.ErrBadRequest)

			return
		}

		if err := h.notifications.MarkMessagesRead(ctx, httphelper.CurrentUserProfile(ctx).SteamID, request.MessageIDs); err != nil && !errors.Is(err, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to mark all notifications read", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onDeleteAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := h.notifications.DeleteAll(ctx, httphelper.CurrentUserProfile(ctx).SteamID); err != nil && !errors.Is(err, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete all notifications", log.ErrAttr(err))

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
			httphelper.HandleErrs(ctx, domain.ErrBadRequest)

			return
		}

		if err := h.notifications.DeleteMessages(ctx, httphelper.CurrentUserProfile(ctx).SteamID, request.MessageIDs); err != nil && !errors.Is(err, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete notifications", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h notificationHandler) onNotifications() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		notifications, err := h.notifications.GetPersonNotifications(ctx, httphelper.CurrentUserProfile(ctx).SteamID)
		if err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				ctx.JSON(http.StatusOK, []domain.UserNotification{})

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get personal notifications", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, notifications)
	}
}
