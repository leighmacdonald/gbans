package anticheat

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type antiCheatHandler struct {
	anticheat AntiCheat
}

func NewAnticheatHandler(engine *gin.Engine, authenticator httphelper.Authenticator, anticheat AntiCheat) {
	handler := &antiCheatHandler{anticheat: anticheat}
	// mod
	modGrp := engine.Group("/api/anticheat")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.PModerator))
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

		detections, errDetections := h.anticheat.BySteamID(ctx, steamID)
		if errDetections != nil && !errors.Is(errDetections, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDetections, httphelper.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDetections, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, detections)
	}
}

func (h antiCheatHandler) query() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var query Query
		if !httphelper.BindQuery(ctx, &query) {
			return
		}

		entries, errEntries := h.anticheat.Query(ctx, query)
		if errEntries != nil && !errors.Is(errEntries, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errEntries, httphelper.ErrInternal)))

			return
		}

		if entries == nil {
			entries = []Entry{}
		}

		ctx.JSON(http.StatusOK, entries)
	}
}
