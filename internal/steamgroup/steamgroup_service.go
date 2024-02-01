package steamgroup

import (
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type SteamgroupHandler struct {
	log *zap.Logger
	bgu domain.BanGroupUsecase
}

func NewSteamgroupHandler(log *zap.Logger, engine *gin.Engine, bgu domain.BanGroupUsecase, ath domain.AuthUsecase) {
	handler := SteamgroupHandler{
		log: log.Named("steamgroup"),
		bgu: bgu,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PUser))
		mod.POST("/api/bans/group/create", handler.onAPIPostBansGroupCreate())
		mod.POST("/api/bans/group", handler.onAPIGetBansGroup())
		mod.DELETE("/api/bans/group/:ban_group_id", handler.onAPIDeleteBansGroup())
		mod.POST("/api/bans/group/:ban_group_id", handler.onAPIPostBansGroupUpdate())
	}
}

func (h SteamgroupHandler) onAPIPostBansGroupCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		GroupID    steamid.GID      `json:"group_id"`
		Duration   string           `json:"duration"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		var existing domain.BanGroup
		if errExist := h.bgu.GetByGID(ctx, req.GroupID, &existing); errExist != nil {
			if !errors.Is(errExist, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}
		}

		var (
			banSteamGroup domain.BanGroup
			sid           = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBanSteamGroup := domain.NewBanSteamGroup(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Note,
			domain.Web,
			req.GroupID,
			"",
			domain.Banned,
			&banSteamGroup,
		); errBanSteamGroup != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Failed to save group ban", zap.Error(errBanSteamGroup))

			return
		}

		if errBan := h.bgu.Ban(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)
	}
}

func (h SteamgroupHandler) onAPIGetBansGroup() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.GroupBansQueryFilter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		banGroups, count, errBans := h.bgu.Get(ctx, req)
		if errBans != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, banGroups))
	}
}

func (h SteamgroupHandler) onAPIDeleteBansGroup() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := httphelper.GetInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInternal)

			return
		}

		var req domain.UnbanRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		var banGroup domain.BanGroup
		if errFetch := h.bgu.GetByID(ctx, groupID, &banGroup); errFetch != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInternal)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := h.bgu.Save(ctx, &banGroup); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0

		ctx.JSON(http.StatusOK, banGroup)
	}
}

func (h SteamgroupHandler) onAPIPostBansGroupUpdate() gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := httphelper.GetInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req apiBanUpdateRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanGroup

		if errExist := h.bgu.GetByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := h.bgu.Save(ctx, &ban); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
