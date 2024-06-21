package steamgroup

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

type steamgroupHandler struct {
	bansGroup domain.BanGroupUsecase
}

func NewSteamgroupHandler(engine *gin.Engine, bgu domain.BanGroupUsecase, ath domain.AuthUsecase) {
	handler := steamgroupHandler{
		bansGroup: bgu,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PUser))
		mod.POST("/api/bans/group/create", handler.onAPIPostBansGroupCreate())
		mod.GET("/api/bans/group", handler.onAPIGetBansGroup())
		mod.DELETE("/api/bans/group/:ban_group_id", handler.onAPIDeleteBansGroup())
		mod.POST("/api/bans/group/:ban_group_id", handler.onAPIPostBansGroupUpdate())
	}
}

func (h steamgroupHandler) onAPIPostBansGroupCreate() gin.HandlerFunc {
	type apiBanGroupRequest struct {
		domain.TargetIDField
		domain.TargetGIDField
		Duration   string    `json:"duration"`
		Note       string    `json:"note"`
		ValidUntil time.Time `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		var req apiBanGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		groupID, errGroupID := req.TargetGroupID(ctx)
		if !errGroupID {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid target group id")

			return
		}

		var existing domain.BanGroup
		if errExist := h.bansGroup.GetByGID(ctx, groupID, &existing); errExist != nil {
			if !errors.Is(errExist, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)
				slog.Error("Tried to ban duplicate group", slog.String("group_id", groupID.String()))

				return
			}
		}

		var (
			banSteamGroup domain.BanGroup
			sid           = httphelper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid duration", log.ErrAttr(errDuration))

			return
		}

		targetID, targetIDOk := req.TargetSteamID(ctx)
		if !targetIDOk {
			httphelper.HandleErrs(ctx, domain.ErrTargetID)
			slog.Warn("Got invalid target id", slog.String("target_id", req.TargetID))

			return
		}

		groupID, groupIDOk := req.TargetGroupID(ctx)
		if !groupIDOk {
			httphelper.HandleErrs(ctx, domain.ErrTargetID)
			slog.Warn("Got invalid group id", slog.String("group_id", req.GroupID))

			return
		}

		if errBanSteamGroup := domain.NewBanSteamGroup(sid, targetID, duration, req.Note, domain.Web, groupID,
			"", domain.Banned, &banSteamGroup); errBanSteamGroup != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to save group ban", log.ErrAttr(errBanSteamGroup))

			return
		}

		if errBan := h.bansGroup.Ban(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to ban group", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)
	}
}

func (h steamgroupHandler) onAPIGetBansGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.GroupBansQueryFilter
		if !httphelper.BindQuery(ctx, &req) {
			return
		}

		banGroups, errBans := h.bansGroup.Get(ctx, req)
		if errBans != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to fetch banGroups", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, banGroups)
	}
}

func (h steamgroupHandler) onAPIDeleteBansGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, groupIDErr := httphelper.GetInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get ban_group_id", log.ErrAttr(groupIDErr))

			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var banGroup domain.BanGroup
		if errFetch := h.bansGroup.GetByID(ctx, groupID, &banGroup); errFetch != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group by id", slog.String("group_id", banGroup.GroupID.String()))

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := h.bansGroup.Save(ctx, &banGroup); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete asn ban", log.ErrAttr(errSave))

			return
		}

		banGroup.BanGroupID = 0

		ctx.JSON(http.StatusOK, banGroup)
	}
}

func (h steamgroupHandler) onAPIPostBansGroupUpdate() gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		domain.TargetIDField
		Note       string    `json:"note"`
		ValidUntil time.Time `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := httphelper.GetInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Warn("Failed to get ban_group_id", log.ErrAttr(banIDErr))

			return
		}

		var req apiBanUpdateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		targetSID, sidValid := req.TargetSteamID(ctx)
		if !sidValid {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid target id", slog.String("target_id", req.TargetID))

			return
		}

		var ban domain.BanGroup

		if errExist := h.bansGroup.GetByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
				slog.Warn("Unknown ban_group_id requested", slog.Int64("ban_group_id", banGroupID))

				return
			}

			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = targetSID

		if errSave := h.bansGroup.Save(ctx, &ban); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save group ban", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
