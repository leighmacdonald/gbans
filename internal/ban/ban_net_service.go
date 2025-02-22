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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, domain.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest,
				errors.Join(errDuration, domain.ErrInvalidBanDuration), "Ban duration invalid"))

			return
		}

		targetID, targetIDOk := req.TargetSteamID(ctx)
		if !targetIDOk {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrTargetID))

			return
		}

		if errBanCIDR := domain.NewBanCIDR(sid, targetID, duration, req.Reason, req.ReasonText, req.Note, domain.Web,
			req.CIDR, domain.Banned, &banCIDR); errBanCIDR != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, errors.Join(errBanCIDR, domain.ErrBadRequest),
				"Could not construct new CIDR ban"))

			return
		}

		if errBan := h.bans.Ban(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, domain.ErrDuplicate,
					"Ban already exists for this user"))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errBan, domain.ErrInternal),
				"Could not save ban"))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, domain.ErrInternal)))

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
			switch {
			case errors.Is(errSave, domain.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, domain.ErrNoResult,
					"CIDR ban with net_id %d does not exist", netID))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
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

		ban, errSave := h.bans.Update(ctx, netID, req)
		if errSave != nil {
			switch {
			case errors.Is(errSave, domain.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, domain.ErrNoResult,
					"CIDR ban with net_id %d does not exist", netID))
			case errors.Is(errSave, domain.ErrInvalidParameter):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrInvalidParameter))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, ban)
		slog.Info("CIDR Ban updated", slog.Int64("net_id", netID))
	}
}
