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
		authed.POST("/api/current_profile/notifications", handler.onAPICurrentProfileNotifications())
	}
}

func (h notificationHandler) onAPICurrentProfileNotifications() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentProfile := httphelper.CurrentUserProfile(ctx)

		var req domain.NotificationQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.SteamID = currentProfile.SteamID.String()

		notifications, count, errNot := h.notifications.GetPersonNotifications(ctx, req)
		if errNot != nil {
			if errors.Is(errNot, domain.ErrNoResult) {
				ctx.JSON(http.StatusOK, []domain.UserNotification{})

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get personal notifications", log.ErrAttr(errNot))

			return
		}

		ctx.JSON(http.StatusOK, domain.LazyResult{
			Count: count,
			Data:  notifications,
		})
	}
}
