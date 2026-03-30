package mge

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type Handler struct {
	mge MGE
}

func NewHandler(engine *gin.Engine, _ httphelper.Authenticator, mge MGE) Handler {
	handler := Handler{
		mge: mge,
	}

	engine.GET("/api/mge/ratings/overall", handler.getRatings())
	engine.GET("/api/mge/history", handler.getHistory())

	return handler
}

func (h Handler) getRatings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindQuery[QueryOpts](ctx)
		if !ok {
			return
		}

		history, count, errChat := h.mge.Query(ctx, req)
		if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(count, history))
	}
}

func (h Handler) getHistory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindQuery[HistoryOpts](ctx)
		if !ok {
			return
		}

		history, count, errChat := h.mge.History(ctx, req)
		if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errChat, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(count, history))
	}
}
