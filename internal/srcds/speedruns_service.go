package srcds

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type speedrunHandler struct {
	speedruns domain.SpeedrunUsecase
	auth      domain.AuthUsecase
}

func NewSpeedrunHandler(engine *gin.Engine, speedruns domain.SpeedrunUsecase, auth domain.AuthUsecase) {
	handler := speedrunHandler{
		speedruns: speedruns,
		auth:      auth,
	}

	guestGroup := engine.Group("/api/speedruns")
	{
		guest := guestGroup.Use(auth.AuthMiddleware(domain.PGuest))
		// Groups
		guest.GET("/overall", handler.getOverall())
		guest.GET("/map", handler.getLeaders())
	}
}

func (s *speedrunHandler) getOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var results []domain.Speedrun
		for i := range 100 {
			sr := domain.Speedrun{
				SpeedrunID:    i + 1,
				MapName:       "",
				PointCaptures: nil,
				Players:       nil,
				Duration:      0,
				PlayerCount:   0,
				HostAddr:      "",
				BotCount:      0,
				CreatedOn:     time.Time{},
				Category:      "",
			}
			results = append(results, sr)
		}

		ctx.JSON(http.StatusOK, results)
	}
}

func (s *speedrunHandler) getLeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{})
	}
}
