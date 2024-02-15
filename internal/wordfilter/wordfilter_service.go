package wordfilter

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wordFilterHandler struct {
	wfu         domain.WordFilterUsecase
	cu          domain.ChatUsecase
	confUsecase domain.ConfigUsecase
}

func NewWordFilterHandler(engine *gin.Engine, confUsecase domain.ConfigUsecase, wfu domain.WordFilterUsecase, cu domain.ChatUsecase, ath domain.AuthUsecase) {
	handler := wordFilterHandler{
		confUsecase: confUsecase,
		wfu:         wfu,
		cu:          cu,
	}

	// editor
	editorGrp := engine.Group("/")
	{
		editor := editorGrp.Use(ath.AuthMiddleware(domain.PUser))
		editor.POST("/api/filters/query", handler.onAPIQueryWordFilters())
		editor.GET("/api/filters/state", handler.onAPIGetWarningState())
		editor.POST("/api/filters", handler.onAPIPostWordFilter())
		editor.DELETE("/api/filters/:word_id", handler.onAPIDeleteWordFilter())
		editor.POST("/api/filter_match", handler.onAPIPostWordMatch())
	}
}

func (h *wordFilterHandler) onAPIQueryWordFilters() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts domain.FiltersQueryFilter
		if !httphelper.Bind(ctx, &opts) {
			return
		}

		words, count, errGetFilters := h.wfu.GetFilters(ctx, opts)
		if errGetFilters != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, words))
	}
}

func (h *wordFilterHandler) onAPIPostWordFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.Filter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if err := httphelper.ErrorHandledWithReturn(ctx, h.wfu.SaveFilter(ctx, httphelper.CurrentUserProfile(ctx), &req)); err != nil {
			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func (h *wordFilterHandler) onAPIDeleteWordFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := httphelper.GetInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		filter, errGet := h.wfu.GetFilterByID(ctx, wordID)
		if errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if errDrop := h.wfu.DropFilter(ctx, &filter); errDrop != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	}
}

func (h *wordFilterHandler) onAPIPostWordMatch() gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	return func(ctx *gin.Context) {
		var req matchRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		words, _, errGetFilters := h.wfu.GetFilters(ctx, domain.FiltersQueryFilter{})
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

func (h *wordFilterHandler) onAPIGetWarningState() gin.HandlerFunc {
	type warningState struct {
		MaxWeight int                  `json:"max_weight"`
		Current   []domain.UserWarning `json:"current"`
	}

	maxWeight := h.confUsecase.Config().Filter.MaxWeight

	return func(ctx *gin.Context) {
		state := h.cu.WarningState()

		outputState := warningState{MaxWeight: maxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
