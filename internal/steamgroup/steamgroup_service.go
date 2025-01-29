package steamgroup

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type steamgroupHandler struct {
	bansGroup domain.BanGroupUsecase
}

func NewHandler(engine *gin.Engine, bgu domain.BanGroupUsecase, ath domain.AuthUsecase) {
	handler := steamgroupHandler{
		bansGroup: bgu,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.Middleware(domain.PModerator))
		mod.POST("/api/bans/group/create", handler.onAPIPostBansGroupCreate())
		mod.GET("/api/bans/group", handler.onAPIGetBansGroup())
		mod.DELETE("/api/bans/group/:ban_group_id", handler.onAPIDeleteBansGroup())
		mod.POST("/api/bans/group/:ban_group_id", handler.onAPIPostBansGroupUpdate())
	}
}

func (h steamgroupHandler) onAPIPostBansGroupCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.RequestBanGroupCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		sourceID := httphelper.CurrentUserProfile(ctx).SteamID

		req.SourceID = sourceID.String()

		ban, errBan := h.bansGroup.Ban(ctx, req)
		if errBan != nil {
			httphelper.HandleErrs(ctx, errBan)
			slog.Error("Failed to ban group", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, ban)
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

		if err := h.bansGroup.Delete(ctx, groupID, req); err != nil {
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to delete asn ban", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h steamgroupHandler) onAPIPostBansGroupUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banGroupID, banIDErr := httphelper.GetInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
			slog.Warn("Failed to get ban_group_id", log.ErrAttr(banIDErr))

			return
		}

		var req domain.RequestBanGroupUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ban, errSave := h.bansGroup.Save(ctx, banGroupID, req)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update group ban", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
