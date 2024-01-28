package service

import (
	"errors"
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/pkg/util"
	"go.uber.org/zap"
)

type WordFilterHandler struct {
	wfu         domain.WordFilterUsecase
	cu          domain.ChatUsecase
	confUsecase domain.ConfigUsecase
	log         *zap.Logger
}

func NewWordFilterHandler(log *zap.Logger, engine *gin.Engine, confUsecase domain.ConfigUsecase, wfu domain.WordFilterUsecase, cu domain.ChatUsecase) {
	handler := WordFilterHandler{
		log:         log.Named("wordfilter"),
		confUsecase: confUsecase,
		wfu:         wfu,
		cu:          cu,
	}

	// editor
	engine.POST("/api/filters/query", handler.onAPIQueryWordFilters())
	engine.GET("/api/filters/state", handler.onAPIGetWarningState())
	engine.POST("/api/filters", handler.onAPIPostWordFilter())
	engine.DELETE("/api/filters/:word_id", handler.onAPIDeleteWordFilter())
	engine.POST("/api/filter_match", handler.onAPIPostWordMatch())
}

func (h *WordFilterHandler) onAPIQueryWordFilters() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts domain.FiltersQueryFilter
		if !http_helper.Bind(ctx, log, &opts) {
			return
		}

		words, count, errGetFilters := h.wfu.GetFilters(ctx, opts)
		if errGetFilters != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, words))
	}
}

func (h *WordFilterHandler) onAPIPostWordFilter() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.Filter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Pattern == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		_, errDur := util.ParseDuration(req.Duration)
		if errDur != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, util.ErrInvalidDuration)

			return
		}

		if req.IsRegex {
			_, compErr := regexp.Compile(req.Pattern)
			if compErr != nil {
				http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidRegex)

				return
			}
		}

		if req.Weight < 1 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidWeight)

			return
		}

		now := time.Now()

		if req.FilterID > 0 {
			var existingFilter domain.Filter
			if errGet := h.wfu.GetFilterByID(ctx, req.FilterID, &existingFilter); errGet != nil {
				if errors.Is(errGet, domain.ErrNoResult) {
					http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

					return
				}

				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}

			existingFilter.UpdatedOn = now
			existingFilter.Pattern = req.Pattern
			existingFilter.IsRegex = req.IsRegex
			existingFilter.IsEnabled = req.IsEnabled
			existingFilter.Action = req.Action
			existingFilter.Duration = req.Duration
			existingFilter.Weight = req.Weight

			if errSave := h.wfu.FilterAdd(ctx, &existingFilter); errSave != nil {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}

			req = existingFilter
		} else {
			profile := http_helper.CurrentUserProfile(ctx)
			newFilter := domain.Filter{
				AuthorID:  profile.SteamID,
				Pattern:   req.Pattern,
				Action:    req.Action,
				Duration:  req.Duration,
				CreatedOn: now,
				UpdatedOn: now,
				IsRegex:   req.IsRegex,
				IsEnabled: req.IsEnabled,
				Weight:    req.Weight,
			}

			if errSave := h.wfu.FilterAdd(ctx, &newFilter); errSave != nil {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}

			req = newFilter
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func (h *WordFilterHandler) onAPIDeleteWordFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := http_helper.GetInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var filter domain.Filter
		if errGet := h.wfu.GetFilterByID(ctx, wordID, &filter); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if errDrop := h.wfu.DropFilter(ctx, &filter); errDrop != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		words, _, errGetFilters := h.wfu.GetFilters(ctx, domain.FiltersQueryFilter{})
		if errGetFilters != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
