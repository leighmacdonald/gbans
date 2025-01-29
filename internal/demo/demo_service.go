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
	demos domain.DemoUsecase
}

func NewHandler(engine *gin.Engine, du domain.DemoUsecase) {
	handler := demoHandler{
		demos: du,
	}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())
	engine.GET("/api/demos/cleanup", handler.onAPIGetCleanup())
}

func (h demoHandler) onAPIGetCleanup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.demos.Cleanup(ctx)

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h demoHandler) onAPIPostDemosQuery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		demos, errDemos := h.demos.GetDemos(ctx)
		if errDemos != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to query demos", log.ErrAttr(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, demos)
	}
}
