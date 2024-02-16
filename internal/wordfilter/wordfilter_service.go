package wordfilter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wordFilterHandler struct {
	filterUsecase domain.WordFilterUsecase
	chatUsecase   domain.ChatUsecase
	confUsecase   domain.ConfigUsecase
}

func NewWordFilterHandler(engine *gin.Engine, confUsecase domain.ConfigUsecase, wfu domain.WordFilterUsecase, cu domain.ChatUsecase, ath domain.AuthUsecase) {
	handler := wordFilterHandler{
		confUsecase:   confUsecase,
		filterUsecase: wfu,
		chatUsecase:   cu,
	}

	// editor
	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(ath.AuthMiddleware(domain.PModerator))
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
		var opts domain.FiltersQueryFilter
		if !httphelper.Bind(ctx, &opts) {
			return
		}

		words, count, errGetFilters := h.filterUsecase.GetFilters(ctx, opts)
		if errGetFilters != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, words))
	}
}

func (h *wordFilterHandler) editFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, wordIDErr := httphelper.GetInt64Param(ctx, "filter_id")
		if wordIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		wordFilter, errEdit := h.filterUsecase.Edit(ctx, httphelper.CurrentUserProfile(ctx), filterID, req)
		if errEdit != nil {
			httphelper.ErrorHandled(ctx, errEdit)

			return
		}

		slog.Info("Edited filter", slog.Int64("filter_id", wordFilter.FilterID))

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) createFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		wordFilter, errCreate := h.filterUsecase.Create(ctx, httphelper.CurrentUserProfile(ctx), req)
		if errCreate != nil {
			httphelper.ErrorHandled(ctx, errCreate)

			return
		}

		slog.Info("Created filter", slog.Int64("filter_id", wordFilter.FilterID))

		ctx.JSON(http.StatusOK, wordFilter)
	}
}

func (h *wordFilterHandler) deleteFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, filterIDErr := httphelper.GetInt64Param(ctx, "filter_id")
		if filterIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		filter, errGet := h.filterUsecase.GetFilterByID(ctx, filterID)
		if errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if errDrop := h.filterUsecase.DropFilter(ctx, filter); errDrop != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		slog.Info("Deleted filter", slog.Int64("id", filter.FilterID))

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

		words, _, errGetFilters := h.filterUsecase.GetFilters(ctx, domain.FiltersQueryFilter{})
		if errGetFilters != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

	maxWeight := h.confUsecase.Config().Filter.MaxWeight

	return func(ctx *gin.Context) {
		state := h.chatUsecase.WarningState()

		outputState := warningState{MaxWeight: maxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
