package person

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

type PersonHandler struct {
	pu            domain.PersonUsecase
	configUsecase domain.ConfigUsecase
	log           *zap.Logger
}

func NewPersonHandler(logger *zap.Logger, engine *gin.Engine, configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase, ath domain.AuthUsecase) {
	handler := &PersonHandler{pu: personUsecase, configUsecase: configUsecase, log: logger.Named("PersonHandler")}

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
		mod.POST("/api/players", handler.onAPISearchPlayers())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(ath.AuthMiddleware(domain.PUser))
		admin.PUT("/api/player/:steam_id/permissions", handler.onAPIPutPlayerPermission())
	}
}

func (h PersonHandler) onAPIPutPlayerPermission() gin.HandlerFunc {
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
		if !httphelper.Bind(ctx, h.log, &req) {
			return
		}

		var person domain.Person
		if errGet := h.pu.GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			h.log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == h.configUsecase.Config().General.Owner {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrPermissionDenied)

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := h.pu.SavePerson(ctx, &person); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			h.log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		h.log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}

func (h PersonHandler) onAPIGetPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		var settings domain.PersonSettings

		if err := h.pu.GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h PersonHandler) onAPIPostPersonSettings() gin.HandlerFunc {
	type settingsUpdateReq struct {
		ForumSignature       string `json:"forum_signature"`
		ForumProfileMessages bool   `json:"forum_profile_messages"`
		StatsHidden          bool   `json:"stats_hidden"`
	}

	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		var req settingsUpdateReq

		if !httphelper.Bind(ctx, h.log, &req) {
			return
		}

		var settings domain.PersonSettings

		if err := h.pu.GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		settings.ForumProfileMessages = req.ForumProfileMessages
		settings.StatsHidden = req.StatsHidden
		settings.ForumSignature = util.SanitizeUGC(req.ForumSignature)

		if err := h.pu.SavePersonSettings(ctx, &settings); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to save person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h PersonHandler) onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		profile := httphelper.CurrentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			h.log.Error("Failed to load user profile",
				zap.Int64("sid64", profile.SteamID.Int64()),
				zap.String("name", profile.Name),
				zap.String("permission_level", profile.PermissionLevel.String()))
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func (h PersonHandler) onAPIProfile() gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	type resp struct {
		Player   *domain.Person        `json:"player"`
		Friends  []steamweb.Friend     `json:"friends"`
		Settings domain.PersonSettings `json:"settings"`
	}

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if errBind := ctx.Bind(&req); errBind != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		person := domain.NewPerson(sid)
		if errGetProfile := h.pu.GetOrCreatePersonBySteamID(requestCtx, sid, &person); errGetProfile != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				h.log.Error("Failed to update player summary", zap.Error(err))
			} else {
				if errSave := h.pu.SavePerson(ctx, &person); errSave != nil {
					h.log.Error("Failed to save person summary", zap.Error(errSave))
				}
			}
		}

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			response.Friends = friendList
		}

		response.Player = &person

		var settings domain.PersonSettings
		if err := h.pu.GetPersonSettings(ctx, sid, &settings); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}

func (h PersonHandler) onAPISearchPlayers() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query domain.PlayerQuery
		if !httphelper.Bind(ctx, log, &query) {
			return
		}

		people, count, errGetPeople := h.pu.GetPeople(ctx, query)
		if errGetPeople != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}
