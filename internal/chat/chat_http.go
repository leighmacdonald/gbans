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
	chat *Chat
}

func NewChatHandler(engine *gin.Engine, chat *Chat, authenticator httphelper.Authenticator) {
	handler := chatHandler{chat: chat}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))
		authed.POST("/api/messages", handler.onAPIQueryMessages())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.GET("/api/message/:person_message_id/context/:padding", handler.onAPIQueryMessageContext())
	}
}

func (h chatHandler) onAPIQueryMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req HistoryQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		messages, errChat := h.chat.QueryChatHistory(ctx, user, req)
		if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func (h chatHandler) onAPIQueryMessageContext() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		messageID, idFound := httphelper.GetInt64Param(ctx, "person_message_id")
		if !idFound {
			return
		}

		padding, paddingFound := httphelper.GetIntParam(ctx, "padding")
		if !paddingFound {
			return
		}

		messages, errQuery := h.chat.GetPersonMessageContext(ctx, messageID, padding)
		if errQuery != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errQuery, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
