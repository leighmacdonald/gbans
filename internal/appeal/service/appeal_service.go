package service

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

type AppealHandler struct {
	appealUsecase  domain.AppealUsecase
	banUsecase     domain.BanUsecase
	configUsecase  domain.ConfigUsecase
	personUsecase  domain.PersonUsecase
	discordUsecase domain.DiscordUsecase
	log            *zap.Logger
}

func NewAppealHandler(logger *zap.Logger, engine *gin.Engine, au domain.AppealUsecase, bu domain.BanUsecase,
	cu domain.ConfigUsecase, pu domain.PersonUsecase, du domain.DiscordUsecase) {
	handler := &AppealHandler{
		appealUsecase:  au,
		banUsecase:     bu,
		configUsecase:  cu,
		personUsecase:  pu,
		discordUsecase: du,
		log:            logger.Named("appeal"),
	}

	//authed
	engine.GET("/api/bans/:ban_id/messages", handler.onAPIGetBanMessages())
	engine.POST("/api/bans/:ban_id/messages", handler.onAPIPostBanMessage())
	engine.POST("/api/bans/message/:ban_message_id", handler.onAPIEditBanMessage())
	engine.DELETE("/api/bans/message/:ban_message_id", handler.onAPIDeleteBanMessage())
}

func (a *AppealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := http_helper.GetInt64Param(ctx, "ban_id")
		if errParam != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, domain.domain.ErrInvalidParameter

			return
		}

		banPerson := domain.NewBannedPerson()
		if errGetBan := a.banUsecase.GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		if !http_helper.CheckPrivilege(ctx, http_helper.http_helper.CurrentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
			return
		}

		banMessages, errGetBanMessages := a.appealUsecase.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (a *AppealHandler) onAPIPostBanMessage() gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := a.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := http_helper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrInvalidParameter

			return
		}

		var req newMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		bannedPerson := domain.NewBannedPerson()
		if errReport := a.banUsecase.GetBanByBanID(ctx, banID, &bannedPerson, true); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			log.Error("Failed to load ban", zap.Error(errReport))

			return
		}

		curUserProfile := http_helper.http_helper.CurrentUserProfile(ctx)
		if bannedPerson.AppealState != domain.Open && curUserProfile.PermissionLevel < domain.PModerator {
			http_helper.http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)
			log.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bannedPerson.BanID), zap.Int64("target_id", bannedPerson.TargetID.Int64()))

			return
		}

		msg := domain.NewBanAppealMessage(banID, curUserProfile.SteamID, req.Message)
		if errSave := a.appealUsecase.SaveBanMessage(ctx, &msg); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		msg.PermissionLevel = curUserProfile.PermissionLevel
		msg.Personaname = curUserProfile.Name
		msg.Avatarhash = curUserProfile.Avatarhash

		ctx.JSON(http.StatusCreated, msg)

		var target domain.Person
		if errTarget := a.personUsecase.GetPersonBySteamID(ctx, bannedPerson.TargetID, &target); errTarget != nil {
			log.Error("Failed to load target", zap.Error(errTarget))

			return
		}

		var source domain.Person
		if errSource := a.personUsecase.GetPersonBySteamID(ctx, bannedPerson.SourceID, &source); errSource != nil {
			log.Error("Failed to load source", zap.Error(errSource))

			return
		}

		a.discordUsecase.SendPayload(domain.ChannelModLog, discord.NewAppealMessage(msg.MessageMD,
			a.configUsecase.ExtURL(bannedPerson.BanSteam), curUserProfile, a.configUsecase.ExtURL(curUserProfile)))
	}
}

func (a *AppealHandler) onAPIEditBanMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := a.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrInvalidParameter

			return
		}

		var existing domain.BanAppealMessage
		if errExist := a.appealUsecase.GetBanMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			return
		}

		curUser := http_helper.http_helper.CurrentUserProfile(ctx)

		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			http_helper.http_helper.ResponseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := a.appealUsecase.SaveBanMessage(ctx, &existing); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		a.discordUsecase.SendPayload(domain.ChannelModLog, discord.EditAppealMessage(existing, req.BodyMD, curUser, a.configUsecase.ExtURL(curUser)))
	}
}

func (a *AppealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := http_helper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrInvalidParameter

			return
		}

		var existing domain.BanAppealMessage
		if errExist := a.appealUsecase.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			return
		}

		curUser := http_helper.http_helper.CurrentUserProfile(ctx)
		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := a.appealUsecase.SaveBanMessage(ctx, &existing); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)
			log.Error("Failed to save appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		a.discordUsecase.SendPayload(domain.ChannelModLog, discord.DeleteAppealMessage(existing, curUser, a.configUsecase.ExtURL(curUser)))
	}
}

func onAPIGetAppeals() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.AppealQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := env.Store().GetAppealsByActivity(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(total, bans))
	}
}
