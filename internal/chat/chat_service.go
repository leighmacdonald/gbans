package chat

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type chatHandler struct {
	chat domain.ChatUsecase
}

func NewHandler(engine *gin.Engine, chat domain.ChatUsecase, authUsecase domain.AuthUsecase) {
	handler := chatHandler{chat: chat}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.Middleware(domain.PUser))
		authed.POST("/api/messages", handler.onAPIQueryMessages())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.Middleware(domain.PModerator))
		mod.GET("/api/message/:person_message_id/context/:padding", handler.onAPIQueryMessageContext())
	}
}

func (h chatHandler) onAPIQueryMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ChatHistoryQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		messages, errChat := h.chat.QueryChatHistory(ctx, httphelper.CurrentUserProfile(ctx), req)
		if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, domain.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errQuery, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
