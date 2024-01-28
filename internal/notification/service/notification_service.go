package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"net/http"
	"runtime"
)

func onAPICurrentProfileNotifications() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentProfile := http_helper.CurrentUserProfile(ctx)

		var req domain.NotificationQuery
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		req.SteamID = currentProfile.SteamID

		notifications, count, errNot := env.Store().GetPersonNotifications(ctx, req)
		if errNot != nil {
			if errors.Is(errNot, errs.ErrNoResult) {
				ctx.JSON(http.StatusOK, []domain.UserNotification{})

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.LazyResult{
			Count: count,
			Data:  notifications,
		})
	}
}
