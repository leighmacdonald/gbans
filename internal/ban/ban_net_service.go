package ban

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type banNetHandler struct {
	bans domain.BanNetUsecase
}

func NewHandlerNet(engine *gin.Engine, bans domain.BanNetUsecase, auth domain.AuthUsecase) {
	handler := banNetHandler{
		bans: bans,
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.GET("/export/bans/valve/network", handler.onAPIExportBansValveIP())
		mod.POST("/api/bans/cidr/create", handler.onAPIPostBansCIDRCreate())
		mod.GET("/api/bans/cidr", handler.onAPIGetBansCIDR())
		mod.DELETE("/api/bans/cidr/:net_id", handler.onAPIDeleteBansCIDR())
		mod.POST("/api/bans/cidr/:net_id", handler.onAPIPostBansCIDRUpdate())
	}
}

func (h banNetHandler) onAPIExportBansValveIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, errBans := h.bans.Get(ctx, domain.CIDRBansQueryFilter{})
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}
			// TODO Shouldn't be cidr?
			entries = append(entries, "addip 0 "+ban.CIDR)
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func (h banNetHandler) onAPIPostBansCIDRCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.RequestBanCIDRCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var (
			banCIDR domain.BanCIDR
			sid     = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, errDuration))

			return
		}

		targetID, targetIDOk := req.TargetSteamID(ctx)
		if !targetIDOk {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrTargetID))

			return
		}

		if errBanCIDR := domain.NewBanCIDR(sid, targetID, duration, req.Reason, req.ReasonText, req.Note, domain.Web,
			req.CIDR, domain.Banned, &banCIDR); errBanCIDR != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, errBanCIDR))

			return
		}

		if errBan := h.bans.Ban(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusConflict, errBan))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBan))

			slog.Error("Failed to save cidr ban", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)

		slog.Info("New ban created",
			slog.String("steam_id", banCIDR.TargetID.String()),
			slog.Int64("net_id", banCIDR.NetID),
			slog.String("cidr", banCIDR.CIDR))
	}
}

func (h banNetHandler) onAPIGetBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.CIDRBansQueryFilter
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		bans, errBans := h.bans.Get(ctx, req)
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banNetHandler) onAPIDeleteBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netID, idFound := httphelper.GetInt64Param(ctx, "net_id")
		if !idFound {
			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := h.bans.Delete(ctx, netID, req, false); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("CIDR Ban deleted", slog.Int64("net_id", netID))
	}
}

func (h banNetHandler) onAPIPostBansCIDRUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netID, idFound := httphelper.GetInt64Param(ctx, "net_id")
		if !idFound {
			return
		}

		var req domain.RequestBanCIDRUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if ban, errSave := h.bans.Update(ctx, netID, req); errSave != nil {
			if errors.Is(errSave, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNoResult))

				return
			}
			if errors.Is(errSave, domain.ErrInvalidParameter) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrInvalidParameter))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))
		} else {
			ctx.JSON(http.StatusOK, ban)
			slog.Info("CIDR Ban updated", slog.Int64("net_id", netID))
		}
	}
}
