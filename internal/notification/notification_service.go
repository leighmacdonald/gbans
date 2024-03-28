package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type notificationHandler struct {
	nu domain.NotificationUsecase
}

func NewNotificationHandler(engine *gin.Engine, nu domain.NotificationUsecase, ath domain.AuthUsecase) {
	handler := notificationHandler{
		nu: nu,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
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

		notifications, count, errNot := h.nu.GetPersonNotifications(ctx, req)
		if errNot != nil {
			if errors.Is(errNot, domain.ErrNoResult) {
				ctx.JSON(http.StatusOK, []domain.UserNotification{})

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.LazyResult{
			Count: count,
			Data:  notifications,
		})
	}
}
