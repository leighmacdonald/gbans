package appeal

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type appealHandler struct {
	appealUsecase domain.AppealUsecase
}

func NewAppealHandler(engine *gin.Engine, appealUsecase domain.AppealUsecase, authUsecase domain.AuthUsecase) {
	handler := &appealHandler{
		appealUsecase: appealUsecase,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.AuthMiddleware(domain.PUser))
		authed.GET("/api/bans/:ban_id/messages", handler.onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", handler.createBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", handler.editBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", handler.onAPIDeleteBanMessage())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.AuthMiddleware(domain.PModerator))
		mod.POST("/api/appeals", handler.onAPIGetAppeals())
	}
}

func (h *appealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := httphelper.GetInt64Param(ctx, "ban_id")
		if errParam != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusNotFound, domain.ErrInvalidParameter)
			slog.Warn("Got invalid ban_id parameter", log.ErrAttr(errParam), log.HandlerName(2))

			return
		}

		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, httphelper.CurrentUserProfile(ctx), banID)
		if errGetBanMessages != nil {
			httphelper.HandleErrs(ctx, errGetBanMessages)
			slog.Error("Failed to load ban messages", log.ErrAttr(errGetBanMessages), log.HandlerName(2))

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (h *appealHandler) createBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errID := httphelper.GetInt64Param(ctx, "ban_id")
		if errID != nil {
			httphelper.HandleErrs(ctx, errID)
			slog.Warn("Got invalid ban_id parameter", log.ErrAttr(errID), log.HandlerName(2))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errSave := h.appealUsecase.CreateBanMessage(ctx, httphelper.CurrentUserProfile(ctx), banID, req.BodyMD)
		if err := httphelper.HandleErrsReturn(ctx, errSave); err != nil {
			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h *appealHandler) editBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to get ban_message_id", log.ErrAttr(errID), log.HandlerName(2))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.appealUsecase.EditBanMessage(ctx, curUser, reportMessageID, req.BodyMD)
		if errSave != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save ban appeal message", log.ErrAttr(errSave), log.HandlerName(2))

			return
		}

		ctx.JSON(http.StatusOK, msg)
		slog.Info("Appeal message updated", slog.Int64("message_id", reportMessageID))
	}
}

func (h *appealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := httphelper.CurrentUserProfile(ctx)

		banMessageID, errID := httphelper.GetInt64Param(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to get ban_message_id", log.ErrAttr(errID), log.HandlerName(2))

			return
		}

		if err := httphelper.HandleErrsReturn(ctx, h.appealUsecase.DropBanMessage(ctx, curUser, banMessageID)); err != nil {
			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))
	}
}

func (h *appealHandler) onAPIGetAppeals() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.AppealQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bans, errBans := h.appealUsecase.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch appeals", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}
