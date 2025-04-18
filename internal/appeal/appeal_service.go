package appeal

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type appealHandler struct {
	appealUsecase domain.AppealUsecase
}

func NewHandler(engine *gin.Engine, appealUsecase domain.AppealUsecase, authUsecase domain.AuthUsecase) {
	handler := &appealHandler{
		appealUsecase: appealUsecase,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.Middleware(domain.PUser))
		authed.GET("/api/bans/:ban_id/messages", handler.onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", handler.createBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", handler.editBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", handler.onAPIDeleteBanMessage())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.Middleware(domain.PModerator))
		mod.POST("/api/appeals", handler.onAPIGetAppeals())
	}
}

func (h *appealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, httphelper.CurrentUserProfile(ctx), banID)
		if errGetBanMessages != nil && !errors.Is(errGetBanMessages, domain.ErrNotFound) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetBanMessages, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func (h *appealHandler) createBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errSave := h.appealUsecase.CreateBanMessage(ctx, httphelper.CurrentUserProfile(ctx), banID, req.BodyMD)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h *appealHandler) editBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, idFound := httphelper.GetInt64Param(ctx, "ban_message_id")
		if !idFound {
			return
		}

		if reportMessageID <= 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, domain.ErrBadRequest,
				"ban_message_id cannot be <= 0"))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.appealUsecase.EditBanMessage(ctx, curUser, reportMessageID, req.BodyMD)
		if errSave != nil {
			switch {
			case errors.Is(errSave, domain.ErrParamInvalid):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, domain.ErrBadRequest,
					"Invalid message body"))
			case errors.Is(errSave, domain.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, domain.ErrPermissionDenied,
					"Not allowed to edit message."))
			case errors.Is(errSave, domain.ErrDuplicate):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, domain.ErrDuplicate,
					"Message cannot be the same."))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal),
					"Could not edit ban message"))
			}

			return
		}

		ctx.JSON(http.StatusOK, msg)
	}
}

func (h *appealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := httphelper.CurrentUserProfile(ctx)

		banMessageID, idFound := httphelper.GetInt64Param(ctx, "ban_message_id")
		if !idFound {
			return
		}

		if banMessageID == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, domain.ErrBadRequest,
				"ban_message_id cannot be <= 0"))

			return
		}

		if err := h.appealUsecase.DropBanMessage(ctx, curUser, banMessageID); err != nil {
			switch {
			case errors.Is(err, domain.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, domain.ErrPermissionDenied,
					"You are not allowed to delete this message."))
			case errors.Is(err, domain.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, domain.ErrNoResult,
					"Message does not exist with id: %d", banMessageID))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal),
					"Could not delete message with id: %d", banMessageID))
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}
