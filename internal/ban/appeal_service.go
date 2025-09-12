package ban

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type appealHandler struct {
	appealUsecase AppealsUsecase
}

func NewAppealHandler(engine *gin.Engine, appealUsecase AppealsUsecase, authUsecase httphelper.Authenticator) {
	handler := &appealHandler{
		appealUsecase: appealUsecase,
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.Middleware(permission.PUser))
		authed.GET("/api/bans/:ban_id/messages", handler.onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", handler.createBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", handler.editBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", handler.onAPIDeleteBanMessage())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.Middleware(permission.PModerator))
		mod.POST("/api/appeals", handler.onAPIGetAppeals())
	}
}

func (h *appealHandler) onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}
		user, _ := session.CurrentUserProfile(ctx)
		banMessages, errGetBanMessages := h.appealUsecase.GetBanMessages(ctx, user, banID)
		if errGetBanMessages != nil && !errors.Is(errGetBanMessages, httphelper.ErrNotFound) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetBanMessages, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

type RequestBanMessage struct {
	BodyMD string `json:"body_md"`
}

func (h *appealHandler) createBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		var req RequestBanMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		msg, errSave := h.appealUsecase.CreateBanMessage(ctx, user, banID, req.BodyMD)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"ban_message_id cannot be <= 0"))

			return
		}

		var req RequestBanMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser, _ := session.CurrentUserProfile(ctx)
		msg, errSave := h.appealUsecase.EditBanMessage(ctx, curUser, reportMessageID, req.BodyMD)
		if errSave != nil {
			switch {
			case errors.Is(errSave, domain.ErrParamInvalid):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
					"Invalid message body"))
			case errors.Is(errSave, permission.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
					"Not allowed to edit message."))
			case errors.Is(errSave, database.ErrDuplicate):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, database.ErrDuplicate,
					"Message cannot be the same."))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
					"Could not edit ban message"))
			}

			return
		}

		ctx.JSON(http.StatusOK, msg)
	}
}

func (h *appealHandler) onAPIDeleteBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser, _ := session.CurrentUserProfile(ctx)

		banMessageID, idFound := httphelper.GetInt64Param(ctx, "ban_message_id")
		if !idFound {
			return
		}

		if banMessageID == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"ban_message_id cannot be <= 0"))

			return
		}

		if err := h.appealUsecase.DropBanMessage(ctx, curUser, banMessageID); err != nil {
			switch {
			case errors.Is(err, permission.ErrPermissionDenied):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
					"You are not allowed to delete this message."))
			case errors.Is(err, database.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, database.ErrNoResult,
					"Message does not exist with id: %d", banMessageID))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal),
					"Could not delete message with id: %d", banMessageID))
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *appealHandler) onAPIGetAppeals() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req AppealQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bans, errBans := h.appealUsecase.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}
