package chat

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type chatHandler struct {
	cu domain.ChatUsecase
}

func NewChatHandler(engine *gin.Engine, usecase domain.ChatUsecase, authUsecase domain.AuthUsecase) {
	handler := chatHandler{cu: usecase}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.AuthMiddleware(domain.PUser))
		authed.POST("/api/messages", handler.onAPIQueryMessages())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.AuthMiddleware(domain.PModerator))
		mod.GET("/api/message/:person_message_id/context/:padding", handler.onAPIQueryMessageContext())
	}
}

func (h chatHandler) onAPIQueryMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ChatHistoryQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		messages, count, errChat := h.cu.QueryChatHistory(ctx, httphelper.CurrentUserProfile(ctx), req)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			slog.Error("Failed to query messages history",
				log.ErrAttr(errChat), slog.String("sid", req.SourceID))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, messages))
	}
}

func (h chatHandler) onAPIQueryMessageContext() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		messageID, errMessageID := httphelper.GetInt64Param(ctx, "person_message_id")
		if errMessageID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Debug("Got invalid person_message_id", log.ErrAttr(errMessageID))

			return
		}

		padding, errPadding := httphelper.GetIntParam(ctx, "padding")
		if errPadding != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			slog.Debug("Got invalid padding", log.ErrAttr(errPadding))

			return
		}

		messages, errQuery := h.cu.GetPersonMessageContext(ctx, messageID, padding)
		if errQuery != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
