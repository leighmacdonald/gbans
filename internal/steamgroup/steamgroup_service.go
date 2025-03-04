package steamgroup

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBan, domain.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, banGroups)
	}
}

func (h steamgroupHandler) onAPIDeleteBansGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetInt64Param(ctx, "ban_group_id")
		if !idFound {
			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if err := h.bansGroup.Delete(ctx, groupID, req); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, errors.Join(err, domain.ErrNotFound)))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h steamgroupHandler) onAPIPostBansGroupUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banGroupID, idFound := httphelper.GetInt64Param(ctx, "ban_group_id")
		if !idFound {
			return
		}

		var req domain.RequestBanGroupUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ban, errSave := h.bansGroup.Save(ctx, banGroupID, req)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
