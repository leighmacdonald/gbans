package ban

import (
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/util"
	"go.uber.org/zap"
)

type banASNHandler struct {
	log           *zap.Logger
	banASNUsecase domain.BanASNUsecase
}

func NewBanASNHandler(logger *zap.Logger, engine *gin.Engine, banASNUsecase domain.BanASNUsecase, ath domain.AuthUsecase) {
	handler := banASNHandler{
		log:           logger.Named("ban_asn"),
		banASNUsecase: banASNUsecase,
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/bans/asn/create", handler.onAPIPostBansASNCreate())
		mod.POST("/api/bans/asn", handler.onAPIGetBansASN())
		mod.DELETE("/api/bans/asn/:asn_id", handler.onAPIDeleteBansASN())
		mod.POST("/api/bans/asn/:asn_id", handler.onAPIPostBansASNUpdate())
	}
}

func (h banASNHandler) onAPIPostBansASNCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ASNum      int64            `json:"as_num"`
		Duration   string           `json:"duration"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		var (
			banASN domain.BanASN
			sid    = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBanSteamGroup := domain.NewBanASN(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			domain.Web,
			req.ASNum,
			domain.Banned,
			&banASN,
		); errBanSteamGroup != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.banASNUsecase.Ban(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to save asn ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func (h banASNHandler) onAPIGetBansASN() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ASNBansQueryFilter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		bansASN, count, errBans := h.banASNUsecase.Get(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, bansASN))
	}
}

func (h banASNHandler) onAPIDeleteBansASN() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := httphelper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.UnbanRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		var banAsn domain.BanASN
		if errFetch := h.banASNUsecase.GetByASN(ctx, asnID, &banAsn); errFetch != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := h.banASNUsecase.Save(ctx, &banAsn); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func (h banASNHandler) onAPIPostBansASNUpdate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := httphelper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanASN
		if errBan := h.banASNUsecase.GetByASN(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req apiBanRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if ban.Reason == domain.Custom && req.ReasonText == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := h.banASNUsecase.Save(ctx, &ban); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
