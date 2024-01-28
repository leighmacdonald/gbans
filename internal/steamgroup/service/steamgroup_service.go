package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"time"
)

func onAPIPostBansGroupCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		GroupID    steamid.GID      `json:"group_id"`
		Duration   string           `json:"duration"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var existing domain.BanGroup
		if errExist := env.Store().GetBanGroup(ctx, req.GroupID, &existing); errExist != nil {
			if !errors.Is(errExist, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}
		}

		var (
			banSteamGroup domain.BanGroup
			sid           = http_helper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

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
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Failed to save group ban", zap.Error(errBanSteamGroup))

			return
		}

		if errBan := env.BanSteamGroup(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, errs.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)
	}
}

func onAPIGetBansGroup() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.GroupBansQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		banGroups, count, errBans := env.Store().GetBanGroups(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, banGroups))
	}
}

func onAPIDeleteBansGroup() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := http_helper.GetInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInternal)

			return
		}

		var req apiUnbanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var banGroup domain.BanGroup
		if errFetch := env.Store().GetBanGroupByID(ctx, groupID, &banGroup); errFetch != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInternal)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := env.Store().SaveBanGroup(ctx, &banGroup); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0

		ctx.JSON(http.StatusOK, banGroup)
	}
}

func onAPIPostBansGroupUpdate() gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := http_helper.GetInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var req apiBanUpdateRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var ban domain.BanGroup

		if errExist := env.Store().GetBanGroupByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := env.Store().SaveBanGroup(ctx, &ban); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
