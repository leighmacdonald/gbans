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

func NewBanNetHandler(engine *gin.Engine, bans domain.BanNetUsecase, auth domain.AuthUsecase) {
	handler := banNetHandler{
		bans: bans,
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
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
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Received invalid duration", log.ErrAttr(errDuration))

			return
		}

		targetID, targetIDOk := req.TargetSteamID(ctx)
		if !targetIDOk {
			httphelper.HandleErrs(ctx, domain.ErrTargetID)
			slog.Warn("Got invalid target steam id", slog.String("target_id", req.TargetID))

			return
		}

		if errBanCIDR := domain.NewBanCIDR(sid, targetID, duration, req.Reason, req.ReasonText, req.Note, domain.Web,
			req.CIDR, domain.Banned, &banCIDR); errBanCIDR != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			slog.Warn("Failed to create new ban cidr", log.ErrAttr(errBanCIDR))

			return
		}

		if errBan := h.bans.Ban(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseApiErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
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
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch cidr bans", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banNetHandler) onAPIDeleteBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netID, netIDErr := httphelper.GetInt64Param(ctx, "net_id")
		if netIDErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if errSave := h.bans.Delete(ctx, netID, req, false); errSave != nil {
			httphelper.HandleErrs(ctx, errSave)
			slog.Error("Failed to delete cidr ban", log.ErrAttr(errSave), slog.Int64("net_id", netID))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("CIDR Ban deleted", slog.Int64("net_id", netID))
	}
}

func (h banNetHandler) onAPIPostBansCIDRUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netID, banIDErr := httphelper.GetInt64Param(ctx, "net_id")
		if banIDErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.RequestBanCIDRUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if ban, errSave := h.bans.Update(ctx, netID, req); errSave != nil {
			if errors.Is(errSave, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}
			if errors.Is(errSave, domain.ErrInvalidParameter) {
				httphelper.HandleErrBadRequest(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update cidr ban", log.ErrAttr(errSave), slog.Int64("net_id", netID))
		} else {
			ctx.JSON(http.StatusOK, ban)
			slog.Info("CIDR Ban updated", slog.Int64("net_id", netID))
		}
	}
}
