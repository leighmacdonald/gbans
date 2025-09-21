package chat

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wordFilterHandler struct {
	filters WordFilters
	chat    *Chat
	config  *config.Configuration
}

func NewWordFilterHandler(engine *gin.Engine, config *config.Configuration, wordFilters WordFilters, chat *Chat, auth httphelper.Authenticator) {
	handler := wordFilterHandler{
		config:  config,
		filters: wordFilters,
		chat:    chat,
	}

	// editor
	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(auth.Middleware(permission.Moderator))
		mod.GET("/api/filters", handler.queryFilters())
		mod.GET("/api/filters/state", handler.filterStates())
		mod.POST("/api/filters", handler.createFilter())
		mod.POST("/api/filters/:filter_id", handler.editFilter())
		mod.DELETE("/api/filters/:filter_id", handler.deleteFilter())
		mod.POST("/api/filter_match", handler.checkFilter())
	}
}

func (h *wordFilterHandler) queryFilters() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		words, errGetFilters := h.filters.GetFilters(ctx)
		if errGetFilters != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetFilters, httphelper.ErrInternal)))

			return
		}

		if words == nil {
			words = []Filter{}
		}

		ctx.JSON(http.StatusOK, words)
	}
}

func (h *wordFilterHandler) editFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, idFound := httphelper.GetInt64Param(ctx, "filter_id")
		if !idFound {
			return
		}

		var req Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		wordFilter, errEdit := h.filters.Edit(ctx, user, filterID, req)
		if errEdit != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errEdit, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) createFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		wordFilter, errCreate := h.filters.Create(ctx, user, req)
		if errCreate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errCreate, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) deleteFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, idFound := httphelper.GetInt64Param(ctx, "filter_id")
		if !idFound {
			return
		}

		if errDrop := h.filters.DropFilter(ctx, filterID); errDrop != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errDrop))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *wordFilterHandler) checkFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req httphelper.RequestQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if matches := h.filters.Check(req.Query); matches == nil {
			ctx.JSON(http.StatusOK, []Filter{})
		} else {
			ctx.JSON(http.StatusOK, matches)
		}
	}
}

func (h *wordFilterHandler) filterStates() gin.HandlerFunc {
	type warningState struct {
		MaxWeight int           `json:"max_weight"`
		Current   []UserWarning `json:"current"`
	}

	return func(ctx *gin.Context) {
		state := h.chat.WarningState()
		outputState := warningState{MaxWeight: h.config.Config().Filters.MaxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		if outputState.Current == nil {
			outputState.Current = []UserWarning{}
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
