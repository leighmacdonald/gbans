package match

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type matchHandler struct {
	matches domain.MatchUsecase
	servers domain.ServersUsecase
	config  domain.ConfigUsecase
}

// todo move data updaters to repository.
func NewHandler(ctx context.Context, engine *gin.Engine, matches domain.MatchUsecase, servers domain.ServersUsecase,
	auth domain.AuthUsecase, config domain.ConfigUsecase,
) {
	handler := matchHandler{matches: matches, servers: servers, config: config}

	engine.GET("/api/stats/map", handler.onAPIGetMapUsage())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.Middleware(domain.PUser))
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
	updater := NewDataUpdater(time.Minute*10, func() ([]domain.WeaponsOverallResult, error) {
		weaponStats, errUpdate := h.matches.WeaponsOverall(ctx)
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

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetsStatsWeapon() gin.HandlerFunc {
	type resp struct {
		httphelper.LazyResult
		Weapon domain.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, idFound := httphelper.GetIntParam(ctx, "weapon_id")
		if !idFound {
			return
		}

		var weapon domain.Weapon

		errWeapon := h.matches.GetWeaponByID(ctx, weaponID, &weapon)
		if errWeapon != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, domain.ErrNotFound))

			return
		}

		weaponStats, errChat := h.matches.WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, domain.ErrInternal)))

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: httphelper.NewLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
	}
}

func (h matchHandler) onAPIGetStatsPlayersOverall(ctx context.Context) gin.HandlerFunc {
	updater := NewDataUpdater(time.Minute*10, func() ([]domain.PlayerWeaponResult, error) {
		updatedStats, errChat := h.matches.PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			return nil, errors.Join(errChat, domain.ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetStatsHealersOverall(ctx context.Context) gin.HandlerFunc {
	updater := NewDataUpdater(time.Minute*10, func() ([]domain.HealingOverallResult, error) {
		updatedStats, errChat := h.matches.HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			return nil, errors.Join(errChat, domain.ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetPlayerWeaponStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		weaponStats, errChat := h.matches.WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, domain.ErrInternal)))

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func (h matchHandler) onAPIGetPlayerClassStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		classStats, errChat := h.matches.PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, domain.ErrInternal)))

			return
		}

		if classStats == nil {
			classStats = []domain.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(int64(len(classStats)), classStats))
	}
}

func (h matchHandler) onAPIGetPlayerStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		var por domain.PlayerOverallResult
		if errChat := h.matches.PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func (h matchHandler) onAPIGetMapUsage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := h.matches.GetMapUsageStats(ctx)
		if errServers != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errServers, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func (h matchHandler) onAPIGetMatch() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		matchID, idFound := httphelper.GetUUIDParam(ctx, "match_id")
		if !idFound {
			return
		}

		var match domain.MatchResult

		errMatch := h.matches.MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, domain.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, domain.ErrNotFound))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMatch, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

func (h matchHandler) onAPIGetMatches() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.MatchesQueryOpts
		if !httphelper.Bind(ctx, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := httphelper.CurrentUserProfile(ctx)
		if user.PermissionLevel <= domain.PUser {
			targetID, ok := req.TargetSteamID()
			if !ok {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

				return
			}

			if user.SteamID != targetID {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, domain.ErrPermissionDenied))

				return
			}
		}

		matches, totalCount, errMatches := h.matches.Matches(ctx, req)
		if errMatches != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMatches, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(totalCount, matches))
	}
}
