package person

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type personHandler struct {
	persons domain.PersonUsecase
	config  domain.ConfigUsecase
}

func NewHandler(engine *gin.Engine, config domain.ConfigUsecase, persons domain.PersonUsecase, auth domain.AuthUsecase) {
	handler := &personHandler{persons: persons, config: config}

	engine.GET("/api/profile", handler.onAPIProfile())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.Middleware(domain.PUser))
		authed.GET("/api/current_profile", handler.onAPICurrentProfile())
		authed.GET("/api/current_profile/settings", handler.onAPIGetPersonSettings())
		authed.POST("/api/current_profile/settings", handler.onAPIPostPersonSettings())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.POST("/api/players", handler.searchPlayers())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(auth.Middleware(domain.PAdmin))
		admin.PUT("/api/player/:steam_id/permissions", handler.onAPIPutPlayerPermission())
	}
}

func (h personHandler) onAPIPutPlayerPermission() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		var req domain.RequestPermissionLevelUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if err := h.persons.SetPermissionLevel(ctx, nil, steamID, req.PermissionLevel); err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

			return
		}

		person, errPerson := h.persons.GetPersonBySteamID(ctx, nil, steamID)
		if errPerson != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errPerson))

			return
		}

		ctx.JSON(http.StatusOK, person)

		slog.Info("Player permission updated",
			slog.Int64("steam_id", steamID.Int64()),
			slog.String("permissions", req.PermissionLevel.String()))
	}
}

func (h personHandler) onAPIGetPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		settings, err := h.persons.GetPersonSettings(ctx, user.SteamID)
		if err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

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

		settings, err := h.persons.SavePersonSettings(ctx, httphelper.CurrentUserProfile(ctx), req)
		if err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrInvalidSID))

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func (h personHandler) onAPIProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req domain.RequestQuery
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		response, err := h.persons.QueryProfile(requestCtx, req.Query)
		if err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

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

		people, count, errGetPeople := h.persons.GetPeople(ctx, nil, query)
		if errGetPeople != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetPeople))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}
