package service

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

type PersonHandler struct {
	PersonUsecase domain.PersonUsecase
	configUsecase domain.ConfigUsecase
	log           *zap.Logger
}

func NewPersonHandler(logger *zap.Logger, engine *gin.Engine, configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase) {
	handler := &PersonHandler{PersonUsecase: personUsecase, configUsecase: configUsecase, log: logger.Named("PersonHandler")}

	engine.GET("/api/profile", handler.onAPIProfile())

	// authed
	engine.GET("/api/current_profile", handler.onAPICurrentProfile())
	engine.GET("/api/current_profile/settings", handler.onAPIGetPersonSettings())
	engine.POST("/api/current_profile/settings", handler.onAPIPostPersonSettings())

	// admin
	engine.PUT("/api/player/:steam_id/permissions", handler.onAPIPutPlayerPermission())
}

func (h *PersonHandler) onAPIPutPlayerPermission() gin.HandlerFunc {
	type updatePermissionLevel struct {
		PermissionLevel domain.Privilege `json:"permission_level"`
	}

	return func(ctx *gin.Context) {
		steamID, errParam := http_helper.GetSID64Param(ctx, "steam_id")
		if errParam != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		var req updatePermissionLevel
		if !http_helper.Bind(ctx, h.log, &req) {
			return
		}

		var person domain.Person
		if errGet := h.PersonUsecase.GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			h.log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == h.configUsecase.Config().General.Owner {
			http_helper.http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrPermissionDenied)

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := h.PersonUsecase.SavePerson(ctx, &person); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			h.log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		h.log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}

func (h *PersonHandler) onAPIGetPersonSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := http_helper.http_helper.CurrentUserProfile(ctx)

		var settings domain.PersonSettings

		if err := h.PersonUsecase.GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			h.log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h *PersonHandler) onAPIPostPersonSettings() gin.HandlerFunc {
	type settingsUpdateReq struct {
		ForumSignature       string `json:"forum_signature"`
		ForumProfileMessages bool   `json:"forum_profile_messages"`
		StatsHidden          bool   `json:"stats_hidden"`
	}

	return func(ctx *gin.Context) {
		user := http_helper.http_helper.CurrentUserProfile(ctx)

		var req settingsUpdateReq

		if !http_helper.Bind(ctx, h.log, &req) {
			return
		}

		var settings domain.PersonSettings

		if err := h.PersonUsecase.GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			h.log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		settings.ForumProfileMessages = req.ForumProfileMessages
		settings.StatsHidden = req.StatsHidden
		settings.ForumSignature = util.SanitizeUGC(req.ForumSignature)

		if err := h.PersonUsecase.SavePersonSettings(ctx, &settings); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			h.log.Error("Failed to save person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func (h *PersonHandler) onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		profile := http_helper.http_helper.CurrentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			h.log.Error("Failed to load user profile",
				zap.Int64("sid64", profile.SteamID.Int64()),
				zap.String("name", profile.Name),
				zap.String("permission_level", profile.PermissionLevel.String()))
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func (h *PersonHandler) onAPIProfile() gin.HandlerFunc {
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
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		person := domain.NewPerson(sid)
		if errGetProfile := h.PersonUsecase.GetOrCreatePersonBySteamID(requestCtx, sid, &person); errGetProfile != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			h.log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				h.log.Error("Failed to update player summary", zap.Error(err))
			} else {
				if errSave := h.PersonUsecase.SavePerson(ctx, &person); errSave != nil {
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
		if err := h.PersonUsecase.GetPersonSettings(ctx, sid, &settings); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			h.log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPIProfile() gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	type resp struct {
		Player   *domain.Person        `json:"player"`
		Friends  []steamweb.Friend     `json:"friends"`
		Settings domain.PersonSettings `json:"settings"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if errBind := ctx.Bind(&req); errBind != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		person := domain.NewPerson(sid)
		if errGetProfile := env.Store().GetOrCreatePersonBySteamID(requestCtx, sid, &person); errGetProfile != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				log.Error("Failed to update player summary", zap.Error(err))
			} else {
				if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
					log.Error("Failed to save person summary", zap.Error(errSave))
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
		if err := env.Store().GetPersonSettings(ctx, sid, &settings); err != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPISearchPlayers() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query domain.PlayerQuery
		if !http_helper.Bind(ctx, log, &query) {
			return
		}

		people, count, errGetPeople := env.Store().GetPeople(ctx, query)
		if errGetPeople != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, people))
	}
}

func onAPIQueryPersonConnections() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ConnectionHistoryQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		ipHist, totalCount, errIPHist := env.Store().QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, errs.ErrNoResult) {
			log.Error("Failed to query connection history", zap.Error(errIPHist))
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, ipHist))
	}
}
