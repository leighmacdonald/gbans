package service

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
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

	// mod
	engine.POST("/api/appeals", handler.onAPIGetAppeals())
}

func (h *AppealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := http_helper.GetInt64Param(ctx, "ban_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrInvalidParameter)

			return
		}

		banPerson := domain.NewBannedPerson()
		if errGetBan := h.banUsecase.GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if !http_helper.CheckPrivilege(ctx, http_helper.CurrentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
			return
		}

		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (h *AppealHandler) onAPIPostBanMessage() gin.HandlerFunc {

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := http_helper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.NewBanMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		curUserProfile := http_helper.CurrentUserProfile(ctx)

		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, curUserProfile, domain.BanAppealMessage{BanID: banID, MessageMD: req.Message})
		if err := http_helper.ErrorHandled(ctx, errSave); err != nil {
			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h *AppealHandler) onAPIEditBanMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.HandleErrBadRequest(ctx)

			return
		}

		var existing domain.BanAppealMessage
		if err := http_helper.ErrorHandled(ctx, h.appealUsecase.GetBanMessageByID(ctx, reportMessageID, &existing)); err != nil {
			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)

		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, curUser, existing)
		if errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		h.discordUsecase.SendPayload(domain.ChannelModLog, discord.EditAppealMessage(existing, req.BodyMD, curUser, h.configUsecase.ExtURL(curUser)))
	}
}

func (h *AppealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := http_helper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			http_helper.HandleErrBadRequest(ctx)

			return
		}

		var existing domain.BanAppealMessage
		if errExist := h.appealUsecase.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				http_helper.HandleErrNotFound(ctx)

				return
			}

			http_helper.HandleErrInternal(ctx)

			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)
		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		if err := http_helper.ErrorHandled(ctx, h.appealUsecase.DropBanMessage(ctx, curUser, &existing)); err != nil {
			return
		}

		ctx.JSON(http.StatusNoContent, nil)
		log.Info("appeal message deleted")

	}
}

func (h *AppealHandler) onAPIGetAppeals() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.AppealQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := h.appealUsecase.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(total, bans))
	}
}
