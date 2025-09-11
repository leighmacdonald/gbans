package demo

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person/permission"
)

type DemoHandler struct {
	demos DemoUsecase
}

func NewHandler(engine *gin.Engine, du DemoUsecase, authUC httphelper.Authenticator) {
	handler := DemoHandler{
		demos: du,
	}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())

	adminGrp := engine.Group("/")
	{
		mod := adminGrp.Use(authUC.Middleware(permission.PAdmin))
		mod.GET("/api/demos/cleanup", handler.onAPIGetCleanup())
	}
}

func (h DemoHandler) onAPIGetCleanup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.demos.Cleanup(ctx)

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h DemoHandler) onAPIPostDemosQuery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		demos, errDemos := h.demos.GetDemos(ctx)
		if errDemos != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDemos, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, demos)
	}
}
