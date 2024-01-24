package service

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/http_helper"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type PersonHandler struct {
	PersonUsecase domain.PersonUsecase
	log           *zap.Logger
}

func NewPersonHandler(logger *zap.Logger, engine *gin.Engine, personUsecase domain.PersonUsecase) {
	handler := &PersonHandler{PersonUsecase: personUsecase, log: logger.Named("PersonHandler")}

	engine.GET("/api/profile", handler.onAPIProfile())
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
			http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		person := domain.NewPerson(sid)
		if errGetProfile := h.GetOrCreatePersonBySteamID(requestCtx, sid, &person); errGetProfile != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				h.log.Error("Failed to update player summary", zap.Error(err))
			} else {
				if errSave := h.SavePerson(ctx, &person); errSave != nil {
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
		if err := h.GetPersonSettings(ctx, sid, &settings); err != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			h.log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}
