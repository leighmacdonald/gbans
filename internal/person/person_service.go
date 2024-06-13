package person

import (
	"context"
	"errors"
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

func NewPersonHandler(engine *gin.Engine, config domain.ConfigUsecase, persons domain.PersonUsecase, auth domain.AuthUsecase) {
	handler := &personHandler{persons: persons, config: config}

	engine.GET("/api/profile", handler.onAPIProfile())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.GET("/api/current_profile", handler.onAPICurrentProfile())
		authed.GET("/api/current_profile/settings", handler.onAPIGetPersonSettings())
		authed.POST("/api/current_profile/settings", handler.onAPIPostPersonSettings())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
		mod.POST("/api/players", handler.searchPlayers())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(auth.AuthMiddleware(domain.PAdmin))
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
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get steam_id", log.ErrAttr(errParam))

			return
		}

		var req updatePermissionLevel
		if !httphelper.Bind(ctx, &req) {
			return
		}

		person, errPerson := h.persons.GetPersonBySteamID(ctx, steamID)
		if errPerson != nil {
			if errors.Is(errPerson, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load person by steam id", log.ErrAttr(errPerson))

			return
		}

		// todo move logic to usecase
		person.PermissionLevel = req.PermissionLevel

		if errSave := h.persons.SavePerson(ctx, &person); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save person", log.ErrAttr(errSave))

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
			httphelper.ErrorHandled(ctx, err)
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
	type profileQuery struct {
		Query string `form:"query"`
	}

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		response, err := h.persons.QueryProfile(requestCtx, req.Query)
		if err != nil {
			httphelper.ErrorHandled(ctx, err)
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

		people, count, errGetPeople := h.persons.GetPeople(ctx, query)
		if errGetPeople != nil {
			httphelper.ErrorHandled(ctx, errGetPeople)
			slog.Error("Failed to query players", log.ErrAttr(errGetPeople))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}
