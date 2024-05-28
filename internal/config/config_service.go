package config

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"log/slog"
	"net/http"
)

type configHandler struct {
	cu domain.ConfigUsecase
}

func NewConfigHandler(engine *gin.Engine, cu domain.ConfigUsecase) {
	handler := configHandler{cu: cu}

	engine.GET("/api/config", handler.onAPIGetConfig())
	engine.PUT("/api/config", handler.onAPIPutConfig())
}

func (c configHandler) onAPIGetConfig() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		config := c.cu.Config()

		ctx.JSON(http.StatusOK, config)
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
