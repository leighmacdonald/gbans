package anticheat

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type antiCheatHandler struct {
	anticheat domain.AntiCheatUsecase
}

func NewHandler(engine *gin.Engine, auth domain.AuthUsecase, anticheat domain.AntiCheatUsecase) {
	handler := &antiCheatHandler{anticheat: anticheat}
	// mod
	modGrp := engine.Group("/api/anticheat")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.GET("/entries", handler.query())
		mod.GET("/steamid/:steam_id", handler.bySteamID())
		mod.GET("/detection/:detection_type", handler.byDetection())
	}
}

func (h antiCheatHandler) bySteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		detections, errDetections := h.anticheat.DetectionsBySteamID(ctx, steamID)
		if errDetections != nil && !errors.Is(errDetections, domain.ErrNoResult) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, domain.ErrInternal))

			return
		}

		ctx.JSON(http.StatusOK, detections)
	}
}

func (h antiCheatHandler) byDetection() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		detectionType, typeFound := httphelper.GetStringParam(ctx, "detection_type")
		if !typeFound {
			return
		}

		detections, errDetections := h.anticheat.DetectionsByType(ctx, logparse.Detection(detectionType))
		if errDetections != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errDetections))

			return
		}

		ctx.JSON(http.StatusOK, detections)
	}
}

func (h antiCheatHandler) query() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var query domain.AnticheatQuery
		if !httphelper.BindQuery(ctx, &query) {
			return
		}

		entries, errEntries := h.anticheat.Query(ctx, query)
		if errEntries != nil && !errors.Is(errEntries, domain.ErrNoResult) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errEntries))

			return
		}

		if entries == nil {
			entries = []domain.AnticheatEntry{}
		}

		ctx.JSON(http.StatusOK, entries)
	}
}
