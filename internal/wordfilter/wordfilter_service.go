package wordfilter

import (
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

func NewHandler(engine *gin.Engine, config domain.ConfigUsecase, wordFilters domain.WordFilterUsecase, chat domain.ChatUsecase, auth domain.AuthUsecase) {
	handler := wordFilterHandler{
		config:  config,
		filters: wordFilters,
		chat:    chat,
	}

	// editor
	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(auth.Middleware(domain.PModerator))
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
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get query filters", log.ErrAttr(errGetFilters))

			return
		}

		if words == nil {
			words = []domain.Filter{}
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
		slog.Info("Filter updated", slog.Int64("filter_id", wordFilter.FilterID))
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
		slog.Info("Created filter", slog.Int64("filter_id", wordFilter.FilterID))
	}
}

func (h *wordFilterHandler) deleteFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filterID, filterIDErr := httphelper.GetInt64Param(ctx, "filter_id")
		if filterIDErr != nil {
			httphelper.HandleErrs(ctx, filterIDErr)

			return
		}

		if errDrop := h.filters.DropFilter(ctx, filterID); errDrop != nil {
			httphelper.HandleErrs(ctx, domain.ErrFailedFetchBan)
			slog.Error("Failed to drop filter", log.ErrAttr(errDrop))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("Deleted filter", slog.Int64("filter_id", filterID))
	}
}

func (h *wordFilterHandler) checkFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.RequestQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if matches := h.filters.Check(req.Query); matches == nil {
			ctx.JSON(http.StatusOK, []domain.Filter{})
		} else {
			ctx.JSON(http.StatusOK, matches)
		}
	}
}

func (h *wordFilterHandler) filterStates() gin.HandlerFunc {
	type warningState struct {
		MaxWeight int                  `json:"max_weight"`
		Current   []domain.UserWarning `json:"current"`
	}

	return func(ctx *gin.Context) {
		state := h.chat.WarningState()
		outputState := warningState{MaxWeight: h.config.Config().Filters.MaxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		if outputState.Current == nil {
			outputState.Current = []domain.UserWarning{}
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
