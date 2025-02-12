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
		steamID, errParam := httphelper.GetSID64Param(ctx, "steam_id")
		if errParam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get steam_id", log.ErrAttr(errParam))

			return
		}

		var req domain.RequestPermissionLevelUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if err := h.persons.SetPermissionLevel(ctx, nil, steamID, req.PermissionLevel); err != nil {
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to set permission level", log.ErrAttr(err),
				slog.Int("level", int(req.PermissionLevel)), slog.String("steam_id", steamID.String()))

			return
		}

		person, errPerson := h.persons.GetPersonBySteamID(ctx, nil, steamID)
		if errPerson != nil {
			httphelper.HandleErrs(ctx, errParam)
			slog.Error("Failed to load new person", log.ErrAttr(errParam),
				slog.Int("level", int(req.PermissionLevel)), slog.String("steam_id", steamID.String()))

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
			httphelper.HandleErrInternal(ctx)
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

		settings, err := h.persons.SavePersonSettings(ctx, httphelper.CurrentUserProfile(ctx), req)
		if err != nil {
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to save person settings", log.ErrAttr(err))

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
			httphelper.HandleErrNotFound(ctx)

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
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to query profile", log.ErrAttr(err))

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
			httphelper.HandleErrs(ctx, errGetPeople)
			slog.Error("Failed to query players", log.ErrAttr(errGetPeople))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}
