package demo

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type demoHandler struct {
	du domain.DemoUsecase
}

func NewDemoHandler(engine *gin.Engine, du domain.DemoUsecase) {
	handler := demoHandler{
		du: du,
	}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())
}

func (h demoHandler) onAPIPostDemosQuery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.DemoFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		demos, count, errDemos := h.du.GetDemos(ctx, req)
		if errDemos != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to query demos", log.ErrAttr(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, domain.NewLazyResult(count, demos))
	}
}
