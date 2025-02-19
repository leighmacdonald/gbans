package demo

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type demoHandler struct {
	demos domain.DemoUsecase
}

func NewHandler(engine *gin.Engine, du domain.DemoUsecase, auth domain.AuthUsecase) {
	handler := demoHandler{
		demos: du,
	}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())

	adminGrp := engine.Group("/")
	{
		mod := adminGrp.Use(auth.Middleware(domain.PAdmin))
		mod.GET("/api/demos/cleanup", handler.onAPIGetCleanup())
	}
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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, demos)
	}
}
