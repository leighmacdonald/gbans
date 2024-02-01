package match

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type matchHandler struct {
	log *zap.Logger
	mu  domain.MatchUsecase
}

// todo move data updaters to repository.
func NewMatchHandler(ctx context.Context, logger *zap.Logger, engine *gin.Engine, mu domain.MatchUsecase, ath domain.AuthUsecase) {
	handler := matchHandler{log: logger, mu: mu}

	engine.GET("/api/stats/map", handler.onAPIGetMapUsage())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/logs", handler.onAPIGetMatches())
		authed.GET("/api/log/:match_id", handler.onAPIGetMatch())
		authed.GET("/api/stats/weapons", handler.onAPIGetStatsWeaponsOverall(ctx))
		authed.GET("/api/stats/weapon/:weapon_id", handler.onAPIGetsStatsWeapon())
		authed.GET("/api/stats/players", handler.onAPIGetStatsPlayersOverall(ctx))
		authed.GET("/api/stats/healers", handler.onAPIGetStatsHealersOverall(ctx))
		authed.GET("/api/stats/player/:steam_id/weapons", handler.onAPIGetPlayerWeaponStatsOverall())
		authed.GET("/api/stats/player/:steam_id/classes", handler.onAPIGetPlayerClassStatsOverall())
		authed.GET("/api/stats/player/:steam_id/overall", handler.onAPIGetPlayerStatsOverall())
	}
}

func (h matchHandler) onAPIGetStatsWeaponsOverall(ctx context.Context) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.WeaponsOverallResult, error) {
		weaponStats, errUpdate := h.mu.WeaponsOverall(ctx)
		if errUpdate != nil && !errors.Is(errUpdate, domain.ErrNoResult) {
			return nil, errors.Join(errUpdate, domain.ErrDataUpdate)
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		return weaponStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetsStatsWeapon() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type resp struct {
		domain.LazyResult
		Weapon domain.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := httphelper.GetIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var weapon domain.Weapon

		errWeapon := h.mu.GetWeaponByID(ctx, weaponID, &weapon)

		if errWeapon != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		weaponStats, errChat := h.mu.WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			log.Error("Failed to get weapons overall top stats",
				zap.Error(errChat))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: domain.NewLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
	}
}

func (h matchHandler) onAPIGetStatsPlayersOverall(ctx context.Context) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.PlayerWeaponResult, error) {
		updatedStats, errChat := h.mu.PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			return nil, errors.Join(errChat, domain.ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetStatsHealersOverall(ctx context.Context) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.HealingOverallResult, error) {
		updatedStats, errChat := h.mu.HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			return nil, errors.Join(errChat, domain.ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetPlayerWeaponStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		weaponStats, errChat := h.mu.WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			log.Error("Failed to query player weapons stats",
				zap.Error(errChat))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func (h matchHandler) onAPIGetPlayerClassStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		classStats, errChat := h.mu.PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			log.Error("Failed to query player class stats",
				zap.Error(errChat))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if classStats == nil {
			classStats = []domain.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(classStats)), classStats))
	}
}

func (h matchHandler) onAPIGetPlayerStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var por domain.PlayerOverallResult
		if errChat := h.mu.PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			log.Error("Failed to query player stats overall",
				zap.Error(errChat))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func (h matchHandler) onAPIGetMapUsage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := h.mu.GetMapUsageStats(ctx)
		if errServers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func (h matchHandler) onAPIGetMatch() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := httphelper.GetUUIDParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var match domain.MatchResult

		errMatch := h.mu.MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

func (h matchHandler) onAPIGetMatches() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.MatchesQueryOpts
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := httphelper.CurrentUserProfile(ctx)
		if user.PermissionLevel <= domain.PUser {
			if !req.SteamID.Valid() {
				httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

				return
			}

			if user.SteamID != req.SteamID {
				httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

				return
			}
		}

		matches, totalCount, matchesErr := h.mu.Matches(ctx, req)
		if matchesErr != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to perform query", zap.Error(matchesErr))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, matches))
	}
}
