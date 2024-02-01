package appeal

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type AppealHandler struct {
	appealUsecase  domain.AppealUsecase
	banUsecase     domain.BanSteamUsecase
	configUsecase  domain.ConfigUsecase
	personUsecase  domain.PersonUsecase
	discordUsecase domain.DiscordUsecase
	log            *zap.Logger
}

func NewAppealHandler(logger *zap.Logger, engine *gin.Engine, appealUsecase domain.AppealUsecase, banUsecase domain.BanSteamUsecase,
	configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase, discordUsecase domain.DiscordUsecase,
	authUsecase domain.AuthUsecase,
) {
	handler := &AppealHandler{
		appealUsecase:  appealUsecase,
		banUsecase:     banUsecase,
		configUsecase:  configUsecase,
		personUsecase:  personUsecase,
		discordUsecase: discordUsecase,
		log:            logger.Named("appeal"),
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.AuthMiddleware(domain.PUser))
		authed.GET("/api/bans/:ban_id/messages", handler.onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", handler.onAPIPostBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", handler.onAPIEditBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", handler.onAPIDeleteBanMessage())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.AuthMiddleware(domain.PModerator))
		mod.POST("/api/appeals", handler.onAPIGetAppeals())
	}
}

func (h *AppealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := httphelper.GetInt64Param(ctx, "ban_id")
		if errParam != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrInvalidParameter)

			return
		}

		banPerson := domain.NewBannedPerson()
		if errGetBan := h.banUsecase.GetByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if !httphelper.CheckPrivilege(ctx, httphelper.CurrentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
			return
		}

		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (h *AppealHandler) onAPIPostBanMessage() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := httphelper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.NewBanMessage
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		curUserProfile := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, curUserProfile, domain.BanAppealMessage{BanID: banID, MessageMD: req.Message})
		if err := httphelper.ErrorHandled(ctx, errSave); err != nil {
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
		reportMessageID, errID := httphelper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		var existing domain.BanAppealMessage
		if err := httphelper.ErrorHandled(ctx, h.appealUsecase.GetBanMessageByID(ctx, reportMessageID, &existing)); err != nil {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		if !httphelper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, curUser, existing)

		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
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
		banMessageID, errID := httphelper.GetIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		var existing domain.BanAppealMessage
		if errExist := h.appealUsecase.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)

			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)
		if !httphelper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		if err := httphelper.ErrorHandled(ctx, h.appealUsecase.DropBanMessage(ctx, curUser, &existing)); err != nil {
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
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := h.appealUsecase.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(total, bans))
	}
}
