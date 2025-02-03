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

func NewHandler(engine *gin.Engine, speedruns domain.SpeedrunUsecase, auth domain.AuthUsecase, config domain.ConfigUsecase) {
	handler := speedrunHandler{
		speedruns: speedruns,
		auth:      auth,
		config:    config,
	}

	guestGroup := engine.Group("/")
	{
		guest := guestGroup.Use(auth.Middleware(domain.PGuest))
		// Groups
		// guest.GET("/api/speedruns/overall", handler.getOverall())
		guest.GET("/api/speedruns/map", handler.getByMap())
		guest.GET("/api/speedruns/overall/top", handler.getOverallTopN())
		guest.GET("/api/speedruns/overall/recent", handler.getRecentChanges())
		guest.GET("/api/speedruns/byid/:speedrun_id", handler.getSpeedrun())
	}

	srcdsGroup := engine.Group("/")
	{
		server := srcdsGroup.Use(auth.MiddlewareServer())
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

// func (s *speedrunHandler) getOverall() gin.HandlerFunc {
//	return func(ctx *gin.Context) {
//		top, errTop := s.speedruns.TopNOverall(ctx, 3)
//		if errTop != nil {
//			slog.Error("Failed to load top speedruns", errTop)
//			httphelper.HandleErrInternal(ctx)
//
//			return
//		}
//
//		ctx.JSON(http.StatusOK, top)
//	}
// }

func (s *speedrunHandler) getByMap() gin.HandlerFunc {
	type q struct {
		MapName string `schema:"map_name"`
	}

	return func(ctx *gin.Context) {
		var query q
		if !httphelper.BindQuery(ctx, &query) {
			return
		}

		runs, errRuns := s.speedruns.ByMap(ctx, query.MapName)
		if errRuns != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load map speedrun results", log.ErrAttr(errRuns))

			return
		}

		ctx.JSON(http.StatusOK, runs)
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

func (s *speedrunHandler) getRecentChanges() gin.HandlerFunc {
	var query struct {
		Count int `json:"count"`
	}

	return func(ctx *gin.Context) {
		if !httphelper.BindQuery(ctx, &query) {
			return
		}

		top, errTop := s.speedruns.Recent(ctx, query.Count)
		if errTop != nil {
			if errors.Is(errTop, domain.ErrValueOutOfRange) {
				httphelper.HandleErrBadRequest(ctx)
				slog.Warn("Got out of bounds recent speedruns value", log.ErrAttr(errTop))

				return
			}
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load recent speedruns", log.ErrAttr(errTop))

			return
		}

		ctx.JSON(http.StatusOK, top)
	}
}

func (s *speedrunHandler) getOverallTopN() gin.HandlerFunc {
	var query struct {
		Count int `json:"count"`
	}

	return func(ctx *gin.Context) {
		if !httphelper.BindQuery(ctx, &query) {
			return
		}

		top, errTop := s.speedruns.TopNOverall(ctx, query.Count)
		if errTop != nil {
			if errors.Is(errTop, domain.ErrValueOutOfRange) {
				httphelper.HandleErrBadRequest(ctx)
				slog.Warn("Got out of bounds top n overall speedruns value", log.ErrAttr(errTop))

				return
			}
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load top n overall speedruns", log.ErrAttr(errTop))

			return
		}

		ctx.JSON(http.StatusOK, top)
	}
}
