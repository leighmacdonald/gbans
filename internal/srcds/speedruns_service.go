package srcds

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type speedrunHandler struct {
	speedruns domain.SpeedrunUsecase
	auth      domain.AuthUsecase
	config    domain.ConfigUsecase
}

func NewSpeedrunHandler(engine *gin.Engine, speedruns domain.SpeedrunUsecase, auth domain.AuthUsecase, config domain.ConfigUsecase) {
	handler := speedrunHandler{
		speedruns: speedruns,
		auth:      auth,
		config:    config,
	}

	guestGroup := engine.Group("/")
	{
		guest := guestGroup.Use(auth.AuthMiddleware(domain.PGuest))
		// Groups
		guest.GET("/api/speedruns/overall", handler.getOverall())
		guest.GET("/api/speedruns/map", handler.getLeaders())
		guest.GET("/api/speedruns/byid/:speedrun_id", handler.getSpeedrun())
	}

	srcdsGroup := engine.Group("/")
	{
		server := srcdsGroup.Use(auth.AuthServerMiddleWare())
		server.POST("/api/sm/speedruns", handler.postSpeedrun())
	}
}

func (s *speedrunHandler) postSpeedrun() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var sr domain.Speedrun
		if !httphelper.Bind(ctx, &sr) {
			return
		}

		speedrun, errSpeedrun := s.speedruns.Save(ctx, sr)
		if errSpeedrun != nil {
			slog.Error("Failed to create speedrun", log.ErrAttr(errSpeedrun))
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, speedrun)
	}
}

func (s *speedrunHandler) getOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *speedrunHandler) getLeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *speedrunHandler) getSpeedrun() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		speedrunID, errID := httphelper.GetIntParam(ctx, "speedrun_id")
		if errID != nil {
			slog.Error("Failed to get speedrun parameter", log.ErrAttr(errID))
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		speedrun, errSpeedrun := s.speedruns.ByID(ctx, speedrunID)
		if errSpeedrun != nil {
			if errors.Is(errSpeedrun, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			slog.Error("Failed to load speedrun", log.ErrAttr(errSpeedrun))
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, speedrun)
	}
}
