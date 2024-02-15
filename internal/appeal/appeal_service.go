package appeal

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type AppealHandler struct {
	appealUsecase  domain.AppealUsecase
	banUsecase     domain.BanSteamUsecase
	configUsecase  domain.ConfigUsecase
	personUsecase  domain.PersonUsecase
	discordUsecase domain.DiscordUsecase
}

func NewAppealHandler(engine *gin.Engine, appealUsecase domain.AppealUsecase, banUsecase domain.BanSteamUsecase,
	configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase, discordUsecase domain.DiscordUsecase,
	authUsecase domain.AuthUsecase,
) {
	handler := &AppealHandler{
		appealUsecase:  appealUsecase,
		banUsecase:     banUsecase,
		configUsecase:  configUsecase,
		personUsecase:  personUsecase,
		discordUsecase: discordUsecase,
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

		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, httphelper.CurrentUserProfile(ctx), banID)
		if errGetBanMessages != nil {
			_ = httphelper.ErrorHandledWithReturn(ctx, errGetBanMessages)

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (h *AppealHandler) onAPIPostBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errID := httphelper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.NewBanMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, httphelper.CurrentUserProfile(ctx), banID, req.Message)
		if err := httphelper.ErrorHandledWithReturn(ctx, errSave); err != nil {
			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h *AppealHandler) onAPIEditBanMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		var req editMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.appealUsecase.SaveBanMessage(ctx, curUser, reportMessageID, req.BodyMD)

		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save ban appeal message", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h *AppealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := httphelper.CurrentUserProfile(ctx)

		banMessageID, errID := httphelper.GetInt64Param(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			return
		}

		if err := httphelper.ErrorHandledWithReturn(ctx, h.appealUsecase.DropBanMessage(ctx, curUser, banMessageID)); err != nil {
			return
		}

		ctx.JSON(http.StatusNoContent, nil)
		slog.Info("appeal message deleted")
	}
}

func (h *AppealHandler) onAPIGetAppeals() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.AppealQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bans, total, errBans := h.appealUsecase.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch appeals", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(total, bans))
	}
}
