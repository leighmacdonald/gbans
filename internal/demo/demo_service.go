package demo

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type demoHandler struct {
	log *zap.Logger
	du  domain.DemoUsecase
}

func NewDemoHandler(log *zap.Logger, engine *gin.Engine, du domain.DemoUsecase) {
	handler := demoHandler{
		log: log.Named("demo"),
		du:  du,
	}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())
}

func (h demoHandler) onAPIPostDemosQuery() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.DemoFilter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		demos, count, errDemos := h.du.GetDemos(ctx, req)
		if errDemos != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, domain.NewLazyResult(count, demos))
	}
}
