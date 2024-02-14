package chat

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type ChatHandler struct {
	cu  domain.ChatUsecase
	log *zap.Logger
}

func NewChatHandler(log *zap.Logger, engine *gin.Engine, cu domain.ChatUsecase, ath domain.AuthUsecase) {
	handler := ChatHandler{
		cu:  cu,
		log: log.Named("chat"),
	}

	// authed
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/messages", handler.onAPIQueryMessages())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.GET("/api/message/:person_message_id/context/:padding", handler.onAPIQueryMessageContext())
	}
}

func (h ChatHandler) onAPIQueryMessages() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ChatHistoryQueryFilter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		messages, count, errChat := h.cu.QueryChatHistory(ctx, httphelper.CurrentUserProfile(ctx), req)
		if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
			log.Error("Failed to query messages history",
				zap.Error(errChat), zap.String("sid", string(req.SourceID)))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, messages))
	}
}

func (h ChatHandler) onAPIQueryMessageContext() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		messageID, errMessageID := httphelper.GetInt64Param(ctx, "person_message_id")
		if errMessageID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			log.Debug("Got invalid person_message_id", zap.Error(errMessageID))

			return
		}

		padding, errPadding := httphelper.GetIntParam(ctx, "padding")
		if errPadding != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Debug("Got invalid padding", zap.Error(errPadding))

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
