package person

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type personHandler struct {
	*Persons
}

func NewPersonHandler(engine *gin.Engine, authenticator httphelper.Authenticator, persons *Persons) {
	handler := &personHandler{Persons: persons}

	engine.GET("/api/profile", handler.onAPIProfile())
	engine.GET("/api/steam/validate", handler.onSteamValidate())

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))
		authed.GET("/api/current_profile", handler.onAPICurrentProfile())
		authed.GET("/api/current_profile/settings", handler.onAPIGetPersonSettings())
		authed.POST("/api/current_profile/settings", handler.onAPIPostPersonSettings())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.POST("/api/players", handler.searchPlayers())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(authenticator.Middleware(permission.Admin))
		admin.PUT("/api/player/:steam_id/permissions", handler.onAPIPutPlayerPermission())
	}
}

func (h personHandler) onAPIPutPlayerPermission() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		req, ok := httphelper.BindJSON[RequestPermissionLevelUpdate](ctx)
		if !ok {
			return
		}

		player, errPerson := h.BySteamID(ctx, steamID)
		if errPerson != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errPerson, httphelper.ErrInternal)))

			return
		}

		player.PermissionLevel = req.PermissionLevel

		if err := h.Save(ctx, &player); err != nil {
			if errors.Is(err, permission.ErrDenied) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, permission.ErrDenied))

				return
			}
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, player)

		slog.Info("Player permission updated",
			slog.Int64("steam_id", steamID.Int64()),
			slog.String("permissions", req.PermissionLevel.String()))
	}
}

func (h personHandler) onAPIGetPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)

		settings, err := h.GetPersonSettings(ctx, user.GetSteamID())
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h personHandler) onAPIPostPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindJSON[SettingsUpdate](ctx)
		if !ok {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		settings, err := h.SavePersonSettings(ctx, user, req)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h personHandler) onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, errProfile := session.CurrentUserProfile(ctx)
		if errProfile != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, steamid.ErrInvalidSID))

			return
		}

		if sid := user.GetSteamID(); !sid.Valid() {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, steamid.ErrInvalidSID))

			return
		}

		// TODO custom profile query
		ctx.JSON(http.StatusOK, user)
	}
}

type SteamValidateResponse struct {
	SteamID     string `json:"steam_id"`
	Hash        string `json:"hash"`
	Personaname string `json:"personaname"`
}

func (h personHandler) onSteamValidate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req httphelper.RequestQuery
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		response, err := h.QueryProfile(requestCtx, req.Query)
		if err != nil {
			if errors.Is(err, steamid.ErrInvalidSID) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, steamid.ErrInvalidSID))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, SteamValidateResponse{
			SteamID:     response.Player.SteamID.String(),
			Hash:        response.Player.AvatarHash,
			Personaname: response.Player.PersonaName,
		})
	}
}

func (h personHandler) onAPIProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req httphelper.RequestQuery
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		response, err := h.QueryProfile(requestCtx, req.Query)
		if err != nil {
			if errors.Is(err, steamid.ErrInvalidSID) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, steamid.ErrInvalidSID))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, response)
	}
}

func (h personHandler) searchPlayers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, ok := httphelper.BindJSON[Query](ctx)
		if !ok {
			return
		}

		people, errGetPeople := h.GetPeople(ctx, query)
		if errGetPeople != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetPeople, httphelper.ErrInternal)))

			return
		}

		// FIXME
		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(100, people))
	}
}
