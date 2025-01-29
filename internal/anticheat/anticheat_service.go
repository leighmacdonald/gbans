package anticheat

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
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
		mod.GET("/steamid/:steam_id", handler.bySteamID())
		mod.GET("/detection/:detection_type", handler.byDetection())
	}
}

func (h antiCheatHandler) bySteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			slog.Error("Failed to query anticheat logs by steam_id",
				log.ErrAttr(errSteamID))
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		detections, errDetections := h.anticheat.DetectionsBySteamID(ctx, steamID)
		if errDetections != nil {
			slog.Error("Failed to query anticheat logs by steam_id",
				log.ErrAttr(errSteamID), slog.Int64("steam_id", steamID.Int64()))
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, detections)
	}
}

func (h antiCheatHandler) byDetection() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		detectionType, errDetectionType := httphelper.GetStringParam(ctx, "detection_type")
		if errDetectionType != nil {
			slog.Error("Failed to query anticheat logs by detection_type",
				log.ErrAttr(errDetectionType))
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		detections, errDetections := h.anticheat.DetectionsByType(ctx, logparse.Detection(detectionType))
		if errDetections != nil {
			slog.Error("Failed to query anticheat logs by steam_id",
				log.ErrAttr(errDetections), slog.String("detection_type", detectionType))
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, detections)
	}
}
