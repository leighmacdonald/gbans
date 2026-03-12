package chat

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type chatHandler struct {
	*Chat
}

func NewChatHandler(engine *gin.Engine, chat *Chat, authenticator httphelper.Authenticator) {
	handler := chatHandler{Chat: chat}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))
		authed.GET("/api/messages", handler.getMessages())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.GET("/api/message/:person_message_id/context/:padding", handler.getMessageCtx())
	}
}

func (h chatHandler) getMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindQuery[HistoryQueryFilter](ctx)
		if !ok {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		messages, count, errChat := h.QueryChatHistory(ctx, user, req)
		if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(count, messages))
	}
}

func (h chatHandler) getMessageCtx() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		messageID, idFound := httphelper.GetInt64Param(ctx, "person_message_id")
		if !idFound {
			return
		}

		padding, paddingFound := httphelper.GetIntParam(ctx, "padding")
		if !paddingFound {
			return
		}

		messages, errQuery := h.GetPersonMessageContext(ctx, messageID, padding)
		if errQuery != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errQuery, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
