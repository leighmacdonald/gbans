package ban

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"log/slog"
	"net/http"
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
	return func(ctx *gin.Context) {
		var req domain.RequestBanASNCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, errBan := h.banASN.Ban(ctx, req)
		if errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.HandleErrDuplicate(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save asn ban", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, bannedPerson)
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

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := h.banASN.Delete(ctx, asnID, req); errSave != nil {
			httphelper.HandleErrs(ctx, errSave)
			slog.Error("Failed to delete asn ban", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h banASNHandler) onAPIPostBansASNUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		asnID, asnIDErr := httphelper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.RequestBanASNUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ban, errSave := h.banASN.Update(ctx, asnID, req)
		if errSave != nil {
			httphelper.HandleErrs(ctx, errSave)
			slog.Error("Failed to update ASN ban", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
