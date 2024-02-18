package person

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type personHandler struct {
	pu            domain.PersonUsecase
	configUsecase domain.ConfigUsecase
}

func NewPersonHandler(engine *gin.Engine, configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase, ath domain.AuthUsecase) {
	handler := &personHandler{pu: personUsecase, configUsecase: configUsecase}

	engine.GET("/api/profile", handler.onAPIProfile())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.GET("/api/current_profile", handler.onAPICurrentProfile())
		authed.GET("/api/current_profile/settings", handler.onAPIGetPersonSettings())
		authed.POST("/api/current_profile/settings", handler.onAPIPostPersonSettings())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PUser))
		mod.POST("/api/players", handler.searchPlayers())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(ath.AuthMiddleware(domain.PUser))
		admin.PUT("/api/player/:steam_id/permissions", handler.onAPIPutPlayerPermission())
	}
}

func (h personHandler) onAPIPutPlayerPermission() gin.HandlerFunc {
	type updatePermissionLevel struct {
		PermissionLevel domain.Privilege `json:"permission_level"`
	}

	return func(ctx *gin.Context) {
		steamID, errParam := httphelper.GetSID64Param(ctx, "steam_id")
		if errParam != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req updatePermissionLevel
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		slog.Info("Player permission updated",
			slog.Int64("steam_id", steamID.Int64()),
			slog.String("permissions", req.PermissionLevel.String()))
	}
}

func (h personHandler) onAPIGetPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		settings, err := h.pu.GetPersonSettings(ctx, user.SteamID)
		if err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch person settings", log.ErrAttr(err), slog.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h personHandler) onAPIPostPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.PersonSettingsUpdate

		if !httphelper.Bind(ctx, &req) {
			return
		}

		settings, err := h.pu.SavePersonSettings(ctx, httphelper.CurrentUserProfile(ctx), req)
		if err != nil {
			httphelper.ErrorHandled(ctx, err)

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h personHandler) onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		profile := httphelper.CurrentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			slog.Error("Failed to load user profile",
				slog.Int64("sid64", profile.SteamID.Int64()),
				slog.String("name", profile.Name),
				slog.String("permission_level", profile.PermissionLevel.String()))
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func (h personHandler) onAPIProfile() gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if errBind := ctx.Bind(&req); errBind != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		response, err := h.pu.QueryProfile(requestCtx, req.Query)
		if err != nil {
			httphelper.ErrorHandled(ctx, err)

			return
		}

		ctx.JSON(http.StatusOK, response)
	}
}

func (h personHandler) searchPlayers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var query domain.PlayerQuery
		if !httphelper.Bind(ctx, &query) {
			return
		}

		people, count, errGetPeople := h.pu.GetPeople(ctx, query)
		if errGetPeople != nil {
			httphelper.ErrorHandled(ctx, errGetPeople)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}
