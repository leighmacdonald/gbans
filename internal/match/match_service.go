package match

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type matchHandler struct {
	matches domain.MatchUsecase
	servers domain.ServersUsecase
	config  domain.ConfigUsecase
}

// todo move data updaters to repository.
func NewMatchHandler(ctx context.Context, engine *gin.Engine, matches domain.MatchUsecase, servers domain.ServersUsecase,
	auth domain.AuthUsecase, config domain.ConfigUsecase,
) {
	handler := matchHandler{matches: matches, servers: servers, config: config}

	engine.GET("/api/stats/map", handler.onAPIGetMapUsage())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.POST("/api/logs", handler.onAPIGetMatches())
		authed.GET("/api/log/:match_id", handler.onAPIGetMatch())
		authed.GET("/api/stats/weapons", handler.onAPIGetStatsWeaponsOverall(ctx))
		authed.GET("/api/stats/weapon/:weapon_id", handler.onAPIGetsStatsWeapon())
		authed.GET("/api/stats/players", handler.onAPIGetStatsPlayersOverall(ctx))
		authed.GET("/api/stats/healers", handler.onAPIGetStatsHealersOverall(ctx))
		authed.GET("/api/stats/player/:steam_id/weapons", handler.onAPIGetPlayerWeaponStatsOverall())
		authed.GET("/api/stats/player/:steam_id/classes", handler.onAPIGetPlayerClassStatsOverall())
		authed.GET("/api/stats/player/:steam_id/overall", handler.onAPIGetPlayerStatsOverall())
		authed.POST("/api/sm/match/start", handler.onAPIPostMatchStart())
		authed.GET("/api/sm/match/end", handler.onAPIPostMatchEnd())
	}
}

func (h matchHandler) onAPIPostMatchEnd() gin.HandlerFunc {
	type endMatchResponse struct {
		URL string `json:"url"`
	}

	return func(ctx *gin.Context) {
		serverID, errServerID := httphelper.GetIntParam(ctx, "server_id")
		if errServerID != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Warn("Failed to get server_id", log.ErrAttr(errServerID))

			return
		}

		matchUUID, errEnd := h.matches.EndMatch(ctx, serverID)
		if errEnd != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrUnknownServerID)
			slog.Error("Failed to end match", log.ErrAttr(errEnd))

			return
		}

		ctx.JSON(http.StatusOK, endMatchResponse{URL: h.config.ExtURLRaw("/match/%s", matchUUID.String())})
	}
}

func (h matchHandler) onAPIPostMatchStart() gin.HandlerFunc {
	type matchStartRequest struct {
		MapName  string `json:"map_name"`
		DemoName string `json:"demo_name"`
	}

	type matchStartResponse struct {
		MatchID uuid.UUID `json:"match_id"`
	}

	return func(ctx *gin.Context) {
		var req matchStartRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		serverID, errServerID := httphelper.GetIntParam(ctx, "server_id")
		if errServerID != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrUnknownServerID)
			slog.Warn("Failed to get server_id", log.ErrAttr(errServerID))

			return
		}

		server, errServer := h.servers.Server(ctx, serverID)
		if errServer != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrUnknownServerID)
			slog.Error("Failed to get server", log.ErrAttr(errServer))

			return
		}

		matchUUID, errMatch := h.matches.StartMatch(server, req.MapName, req.DemoName)
		if errMatch != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrUnknownServerID)
			slog.Error("Failed to start match", log.ErrAttr(errMatch))

			return
		}

		ctx.JSON(http.StatusOK, matchStartResponse{MatchID: matchUUID})
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

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetsStatsWeapon() gin.HandlerFunc {
	type resp struct {
		domain.LazyResult
		Weapon domain.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := httphelper.GetIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Warn("Failed to get weapon_id", log.ErrAttr(errWeaponID))

			return
		}

		var weapon domain.Weapon

		errWeapon := h.matches.GetWeaponByID(ctx, weaponID, &weapon)
		if errWeapon != nil {
			httphelper.HandleErrNotFound(ctx)
			slog.Error("Failed to get weapon", log.ErrAttr(errWeapon))

			return
		}

		weaponStats, errChat := h.matches.WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get weapons overall top stats", log.ErrAttr(errChat))

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: domain.NewLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
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
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
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
		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(stats)), stats))
	}
}

func (h matchHandler) onAPIGetPlayerWeaponStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get steam_id", log.ErrAttr(errSteamID))

			return
		}

		weaponStats, errChat := h.matches.WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to query player weapons stats", log.ErrAttr(errChat))

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func (h matchHandler) onAPIGetPlayerClassStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get steam_id", log.ErrAttr(errSteamID))

			return
		}

		classStats, errChat := h.matches.PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to query player class stats", log.ErrAttr(errChat))

			return
		}

		if classStats == nil {
			classStats = []domain.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(classStats)), classStats))
	}
}

func (h matchHandler) onAPIGetPlayerStatsOverall() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errSteamID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get steam_id", log.ErrAttr(errSteamID))

			return
		}

		var por domain.PlayerOverallResult
		if errChat := h.matches.PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to query player stats overall", log.ErrAttr(errChat))

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func (h matchHandler) onAPIGetMapUsage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := h.matches.GetMapUsageStats(ctx)
		if errServers != nil {
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func (h matchHandler) onAPIGetMatch() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		matchID, errID := httphelper.GetUUIDParam(ctx, "match_id")
		if errID != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Warn("Failed to get match_id", log.ErrAttr(errID))

			return
		}

		var match domain.MatchResult

		errMatch := h.matches.MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, domain.ErrNoResult) {
				httphelper.ResponseAPIErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get match by id", log.ErrAttr(errMatch))

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
				httphelper.HandleErrBadRequest(ctx)

				return
			}

			if user.SteamID != targetID {
				httphelper.HandleErrPermissionDenied(ctx)

				return
			}
		}

		matches, totalCount, errMatches := h.matches.Matches(ctx, req)
		if errMatches != nil {
			httphelper.HandleErrs(ctx, errMatches)
			slog.Error("Failed to get matches", log.ErrAttr(errMatches))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, matches))
	}
}
