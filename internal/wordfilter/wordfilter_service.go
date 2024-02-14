package wordfilter

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type WordFilterHandler struct {
	wfu         domain.WordFilterUsecase
	cu          domain.ChatUsecase
	confUsecase domain.ConfigUsecase
	log         *zap.Logger
}

func NewWordFilterHandler(log *zap.Logger, engine *gin.Engine, confUsecase domain.ConfigUsecase, wfu domain.WordFilterUsecase, cu domain.ChatUsecase, ath domain.AuthUsecase) {
	handler := WordFilterHandler{
		log:         log.Named("wordfilter"),
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

func (h *WordFilterHandler) onAPIQueryWordFilters() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts domain.FiltersQueryFilter
		if !httphelper.Bind(ctx, log, &opts) {
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

func (h *WordFilterHandler) onAPIPostWordFilter() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.Filter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if err := httphelper.ErrorHandledWithReturn(ctx, h.wfu.SaveFilter(ctx, httphelper.CurrentUserProfile(ctx), &req)); err != nil {
			return
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func (h *WordFilterHandler) onAPIDeleteWordFilter() gin.HandlerFunc {
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

func (h *WordFilterHandler) onAPIPostWordMatch() gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req matchRequest
		if !httphelper.Bind(ctx, log, &req) {
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

func (h *WordFilterHandler) onAPIGetWarningState() gin.HandlerFunc {
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
