package config

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type configHandler struct {
	cu   domain.ConfigUsecase
	auth domain.AuthUsecase
}

func NewConfigHandler(engine *gin.Engine, cu domain.ConfigUsecase, auth domain.AuthUsecase) {
	handler := configHandler{cu: cu, auth: auth}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.AuthMiddleware(domain.PAdmin))
		admin.GET("/api/config", handler.onAPIGetConfig())
		admin.PUT("/api/config", handler.onAPIPutConfig())
	}
}

func (c configHandler) onAPIGetConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, c.cu.Config())
	}
}

func (c configHandler) onAPIPutConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.Config
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := c.cu.Write(ctx, req); errSave != nil {
			slog.Error("Failed to save new config", log.ErrAttr(errSave))
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}
