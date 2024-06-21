package wordfilter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type wordFilterHandler struct {
	filters domain.WordFilterUsecase
	chat    domain.ChatUsecase
	config  domain.ConfigUsecase
}

func NewWordFilterHandler(engine *gin.Engine, config domain.ConfigUsecase, wordFilters domain.WordFilterUsecase, chat domain.ChatUsecase, auth domain.AuthUsecase) {
	handler := wordFilterHandler{
		config:  config,
		filters: wordFilters,
		chat:    chat,
	}

	// editor
	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(auth.AuthMiddleware(domain.PModerator))
		mod.POST("/api/filters/query", handler.queryFilters())
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
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get query filters", log.ErrAttr(errGetFilters))

			return
		}

		ctx.JSON(http.StatusOK, words)
	}
}

func (h *wordFilterHandler) editFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, wordIDErr := httphelper.GetInt64Param(ctx, "filter_id")
		if wordIDErr != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid filter_id", log.ErrAttr(wordIDErr))

			return
		}

		var req domain.Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		wordFilter, errEdit := h.filters.Edit(ctx, httphelper.CurrentUserProfile(ctx), filterID, req)
		if errEdit != nil {
			httphelper.HandleErrs(ctx, errEdit)
			slog.Error("Failed to edit word filter", log.ErrAttr(errEdit))

			return
		}

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) createFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		wordFilter, errCreate := h.filters.Create(ctx, httphelper.CurrentUserProfile(ctx), req)
		if errCreate != nil {
			httphelper.HandleErrs(ctx, errCreate)
			slog.Error("Failed to create word filter", log.ErrAttr(errCreate))

			return
		}

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) deleteFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, filterIDErr := httphelper.GetInt64Param(ctx, "filter_id")
		if filterIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Warn("Failed to get filter_id", log.ErrAttr(filterIDErr))

			return
		}

		filter, errGet := h.filters.GetFilterByID(ctx, filterID)
		if errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get filter", log.ErrAttr(errGet))

			return
		}

		if errDrop := h.filters.DropFilter(ctx, filter); errDrop != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to drop filter", log.ErrAttr(errDrop))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *wordFilterHandler) checkFilter() gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	return func(ctx *gin.Context) {
		var req matchRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		words, errGetFilters := h.filters.GetFilters(ctx)
		if errGetFilters != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get filters", log.ErrAttr(errGetFilters))

			return
		}

		var matches []domain.Filter

		for _, filter := range words {
			if filter.Match(req.Query) {
				matches = append(matches, filter)
			}
		}

		ctx.JSON(http.StatusOK, matches)
	}
}

func (h *wordFilterHandler) filterStates() gin.HandlerFunc {
	type warningState struct {
		MaxWeight int                  `json:"max_weight"`
		Current   []domain.UserWarning `json:"current"`
	}

	maxWeight := h.config.Config().Filters.MaxWeight

	return func(ctx *gin.Context) {
		state := h.chat.WarningState()

		outputState := warningState{MaxWeight: maxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		if outputState.Current == nil {
			outputState.Current = []domain.UserWarning{}
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
