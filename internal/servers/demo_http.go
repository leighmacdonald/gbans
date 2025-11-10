package servers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type DemoHandler struct {
	Demos
}

func NewDemoHandler(engine *gin.Engine, authenticator httphelper.Authenticator, du Demos) {
	handler := DemoHandler{Demos: du}

	engine.POST("/api/demos", handler.onAPIPostDemosQuery())

	adminGrp := engine.Group("/")
	{
		mod := adminGrp.Use(authenticator.Middleware(permission.Admin))
		mod.GET("/api/demos/cleanup", handler.onAPIGetCleanup())
	}
}

func (h DemoHandler) onAPIGetCleanup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.Cleanup(ctx)

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h DemoHandler) onAPIPostDemosQuery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		demos, errDemos := h.GetDemos(ctx)
		if errDemos != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDemos, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, demos)
	}
}
