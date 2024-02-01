package notification

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type notificationHandler struct {
	nu  domain.NotificationUsecase
	log *zap.Logger
}

func NewNotificationHandler(log *zap.Logger, engine *gin.Engine, nu domain.NotificationUsecase, ath domain.AuthUsecase) {
	handler := notificationHandler{
		nu:  nu,
		log: log.Named("notif"),
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/current_profile/notifications", handler.onAPICurrentProfileNotifications())
	}
}

func (h notificationHandler) onAPICurrentProfileNotifications() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentProfile := httphelper.CurrentUserProfile(ctx)

		var req domain.NotificationQuery
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		req.SteamID = currentProfile.SteamID

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
