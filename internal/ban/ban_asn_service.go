package ban

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type banASNHandler struct {
	banASN domain.BanASNUsecase
}

func NewBanASNHandler(engine *gin.Engine, banASNUsecase domain.BanASNUsecase, ath domain.AuthUsecase) {
	handler := banASNHandler{
		banASN: banASNUsecase,
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/bans/asn/create", handler.onAPIPostBansASNCreate())
		mod.GET("/api/bans/asn", handler.onAPIGetBansASN())
		mod.DELETE("/api/bans/asn/:asn_id", handler.onAPIDeleteBansASN())
		mod.POST("/api/bans/asn/:asn_id", handler.onAPIPostBansASNUpdate())
	}
}

func (h banASNHandler) onAPIPostBansASNCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		domain.TargetIDField
		Note       string        `json:"note"`
		Reason     domain.Reason `json:"reason"`
		ReasonText string        `json:"reason_text"`
		ASNum      int64         `json:"as_num"`
		Duration   string        `json:"duration"`
		ValidUntil time.Time     `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var (
			banASN domain.BanASN
			sid    = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		targetID, targetIDOk := req.TargetSteamID(ctx)
		if !targetIDOk {
			httphelper.ErrorHandled(ctx, domain.ErrTargetID)

			return
		}

		if errBanSteamGroup := domain.NewBanASN(sid, targetID, duration, req.Reason, req.ReasonText, req.Note, domain.Web,
			req.ASNum, domain.Banned, &banASN); errBanSteamGroup != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.banASN.Ban(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			slog.Error("Failed to save asn ban", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func (h banASNHandler) onAPIGetBansASN() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ASNBansQueryFilter
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		bansASN, errBans := h.banASN.Get(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch banASN", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, bansASN)
	}
}

func (h banASNHandler) onAPIDeleteBansASN() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		asnID, asnIDErr := httphelper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.UnbanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var banAsn domain.BanASN
		if errFetch := h.banASN.GetByID(ctx, asnID, &banAsn); errFetch != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := h.banASN.Save(ctx, &banAsn); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to delete asn ban", log.ErrAttr(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func (h banASNHandler) onAPIPostBansASNUpdate() gin.HandlerFunc {
	type apiBanRequest struct {
		domain.TargetIDField
		ASNum      int64         `json:"as_num"`
		Note       string        `json:"note"`
		Reason     domain.Reason `json:"reason"`
		ReasonText string        `json:"reason_text"`
		ValidUntil time.Time     `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		asnID, asnIDErr := httphelper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanASN
		if errBan := h.banASN.GetByID(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req apiBanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if ban.Reason == domain.Custom && req.ReasonText == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		targetID, targetIDOK := req.TargetSteamID(ctx)
		if !targetIDOK {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ASNum = req.ASNum
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = targetID
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := h.banASN.Save(ctx, &ban); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
