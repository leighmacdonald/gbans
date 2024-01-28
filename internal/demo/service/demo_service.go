package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

func onAPIPostDemosQuery() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.DemoFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		demos, count, errDemos := env.Store().GetDemos(ctx, req)
		if errDemos != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, domain.NewLazyResult(count, demos))
	}
}
