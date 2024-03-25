package ban

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
)

type banNetHandler struct {
	banNetUsecase domain.BanNetUsecase
}

func NewBanNetHandler(engine *gin.Engine, banNetUsecase domain.BanNetUsecase, ath domain.AuthUsecase) {
	handler := banNetHandler{
		banNetUsecase: banNetUsecase,
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.GET("/export/bans/valve/network", handler.onAPIExportBansValveIP())
		mod.POST("/api/bans/cidr/create", handler.onAPIPostBansCIDRCreate())
		mod.POST("/api/bans/cidr", handler.onAPIGetBansCIDR())
		mod.DELETE("/api/bans/cidr/:net_id", handler.onAPIDeleteBansCIDR())
		mod.POST("/api/bans/cidr/:net_id", handler.onAPIPostBansCIDRUpdate())
	}
}

func (h banNetHandler) onAPIExportBansValveIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := h.banNetUsecase.Get(ctx, domain.CIDRBansQueryFilter{})
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}
			// TODO Shouldn't be cidr?
			entries = append(entries, fmt.Sprintf("addip 0 %s", ban.CIDR))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func (h banNetHandler) onAPIPostBansCIDRCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   steamid.SteamID `json:"target_id"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     domain.Reason   `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var (
			banCIDR domain.BanCIDR
			sid     = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBanCIDR := domain.NewBanCIDR(sid, req.TargetID, duration, req.Reason, req.ReasonText, req.Note, domain.Web,
			req.CIDR, domain.Banned, &banCIDR); errBanCIDR != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.banNetUsecase.Ban(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save cidr ban", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)
	}
}

func (h banNetHandler) onAPIGetBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.CIDRBansQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bans, count, errBans := h.banNetUsecase.Get(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch cidr bans", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, bans))
	}
}

func (h banNetHandler) onAPIDeleteBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netID, netIDErr := httphelper.GetInt64Param(ctx, "net_id")
		if netIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.UnbanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var banCidr domain.BanCIDR
		if errFetch := h.banNetUsecase.GetByID(ctx, netID, &banCidr); errFetch != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := h.banNetUsecase.Save(ctx, &banCidr); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to delete cidr ban", log.ErrAttr(errSave))

			return
		}

		banCidr.NetID = 0

		ctx.JSON(http.StatusOK, banCidr)
	}
}

func (h banNetHandler) onAPIPostBansCIDRUpdate() gin.HandlerFunc {
	type apiUpdateBanRequest struct {
		TargetID   steamid.SteamID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     domain.Reason   `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		netID, banIDErr := httphelper.GetInt64Param(ctx, "net_id")
		if banIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanCIDR

		if errBan := h.banNetUsecase.GetByID(ctx, netID, &ban); errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req apiUpdateBanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if !req.TargetID.Valid() {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		if req.Reason == domain.Custom && req.ReasonText == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(req.CIDR)
		if errParseCIDR != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText
		ban.CIDR = ipNet.String()
		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = req.TargetID

		if errSave := h.banNetUsecase.Save(ctx, &ban); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
