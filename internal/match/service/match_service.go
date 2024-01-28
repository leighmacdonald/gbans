package service

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/pkg/util"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"time"
)

type MatchHandler struct {
	log *zap.Logger
	mu domain.MatchUsecase
}

func NewMatchHandler(logger *zap.Logger, engine *gin.Engine, mu domain.MatchUsecase) {
	handler := MatchHandler{log: logger, mu: mu}

	//authed
	engine.POST("/api/logs", handler.onAPIGetMatches())
	engine.GET("/api/log/:match_id", handler.onAPIGetMatch())
	engine.GET("/api/stats/weapons", handler.onAPIGetStatsWeaponsOverall())
	engine.GET("/api/stats/weapon/:weapon_id", handler.onAPIGetsStatsWeapon())
	engine.GET("/api/stats/players", handler.onAPIGetStatsPlayersOverall())
	engine.GET("/api/stats/healers",handler.onAPIGetStatsHealersOverall())
	engine.GET("/api/stats/player/:steam_id/weapons", handler.onAPIGetPlayerWeaponStatsOverall())
	engine.GET("/api/stats/player/:steam_id/classes", handler.onAPIGetPlayerClassStatsOverall())
	engine.GET("/api/stats/player/:steam_id/overall", handler.onAPIGetPlayerStatsOverall())
}


func (h MatchHandler) onAPIGetStatsWeaponsOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := util.NewDataUpdater(log, time.Minute*10, func() ([]domain.WeaponsOverallResult, error) {
		weaponStats, errUpdate := h.mu.WeaponsOverall(ctx)
		if errUpdate != nil && !errors.Is(errUpdate, errs.ErrNoResult) {
			return nil, errors.Join(errUpdate, ErrDataUpdate)
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

func (h MatchHandler) onAPIGetsStatsWeapon() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type resp struct {
		domain.LazyResult
		Weapon domain.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := getIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var weapon domain.Weapon

		errWeapon := env.Store().GetWeaponByID(ctx, weaponID, &weapon)

		if errWeapon != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		weaponStats, errChat := env.Store().WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to get weapons overall top stats",
				zap.Error(errChat))
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: domain.NewLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
	}
}

func (h MatchHandler) onAPIGetStatsPlayersOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.PlayerWeaponResult, error) {
		updatedStats, errChat := env.Store().PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			return nil, errors.Join(errChat, ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h MatchHandler) onAPIGetStatsHealersOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.HealingOverallResult, error) {
		updatedStats, errChat := env.Store().HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			return nil, errors.Join(errChat, ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h MatchHandler) onAPIGetPlayerWeaponStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		weaponStats, errChat := env.Store().WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player weapons stats",
				zap.Error(errChat))
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func (h MatchHandler) onAPIGetPlayerClassStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		classStats, errChat := env.Store().PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player class stats",
				zap.Error(errChat))
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if classStats == nil {
			classStats = []domain.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(classStats)), classStats))
	}
}

func (h MatchHandler) onAPIGetPlayerStatsOverall() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var por domain.PlayerOverallResult
		if errChat := env.Store().PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player stats overall",
				zap.Error(errChat))
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func (h MatchHandler) onAPIGetMapUsage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := env.Store().GetMapUsageStats(ctx)
		if errServers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func (h MatchHandler) onAPIGetMatch() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := getUUIDParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var match domain.MatchResult

		errMatch := env.Store().MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

func(h MatchHandler)  onAPIGetMatches() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.MatchesQueryOpts
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := http_helper.CurrentUserProfile(ctx)
		if user.PermissionLevel <= domain.PUser {
			if !req.SteamID.Valid() {
				http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

				return
			}

			if user.SteamID != req.SteamID {
				http_helper.ResponseErr(ctx, http.StatusForbidden, errPermissionDenied)

				return
			}
		}

		matches, totalCount, matchesErr := env.Store().Matches(ctx, req)
		if matchesErr != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to perform query", zap.Error(matchesErr))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, matches))
	}
}
