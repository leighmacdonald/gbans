package network

import (
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type blocklistHandler struct {
	blocklists BlocklistUsecase
	networks   NetworkUsecase
}

func NewBlocklistHandler(engine *gin.Engine, bu BlocklistUsecase, nu NetworkUsecase, ath httphelper.Authenticator) {
	handler := blocklistHandler{
		blocklists: bu,
		networks:   nu,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.Middleware(permission.PModerator))
		mod.GET("/api/block_list/sources", handler.onAPIGetBlockListSources())

		mod.GET("/api/block_list/whitelist/ip", handler.onAPIWhitelistIPs())
		mod.POST("/api/block_list/whitelist/ip", handler.onAPICreateWhitelistIP())
		mod.DELETE("/api/block_list/whitelist/ip/:cidr_block_whitelist_id", handler.onAPIDeleteBlockListWhitelist())
		mod.POST("/api/block_list/whitelist/ip/:cidr_block_whitelist_id", handler.onAPIUpdateWhitelistIP())

		mod.GET("/api/block_list/whitelist/steam", handler.onAPIWhitelistSteam())
		mod.DELETE("/api/block_list/whitelist/steam/:steam_id", handler.onAPIDeleteWhitelistSteam())
		mod.POST("/api/block_list/whitelist/steam", handler.onAPICreateWhitelistSteam())

		mod.POST("/api/block_list/checker", handler.onAPIPostBlocklistCheck())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(ath.Middleware(permission.PAdmin))
		admin.POST("/api/block_list/sources", handler.onAPIPostBlockListCreate())
		admin.POST("/api/block_list/sources/:cidr_block_source_id", handler.onAPIPostBlockListUpdate())
		admin.DELETE("/api/block_list/sources/:cidr_block_source_id", handler.onAPIDeleteBlockList())
	}
}

type (
	CIDRBlockWhitelistExport struct {
		CIDRBlockWhitelistID int       `json:"cidr_block_whitelist_id"`
		Address              string    `json:"address"`
		CreatedOn            time.Time `json:"created_on"`
		UpdatedOn            time.Time `json:"updated_on"`
	}
)

func (b *blocklistHandler) onAPIWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.blocklists.GetSteamBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errWl, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, whiteLists)
	}
}

func (b *blocklistHandler) onAPICreateWhitelistSteam() gin.HandlerFunc {
	type createRequest struct {
		domain.SteamIDField
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		steamID, ok := req.SteamID(ctx)
		if !ok {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrInvalidSID))

			return
		}

		whitelist, errSave := b.blocklists.CreateSteamBlockWhitelists(ctx, steamID)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, whitelist)
	}
}

func (b *blocklistHandler) onAPIDeleteBlockList() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sourceID, idFound := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if !idFound {
			return
		}

		if err := b.blocklists.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal),
				"Could not delete blocklist source: %d", sourceID))

			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (b *blocklistHandler) onAPIWhitelistIPs() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.blocklists.GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errWl, httphelper.ErrInternal),
				"Could not load whitelist"))

			return
		}

		var wlExported []CIDRBlockWhitelistExport
		for _, whitelist := range whiteLists {
			wlExported = append(wlExported, CIDRBlockWhitelistExport{
				CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
				Address:              whitelist.Address.String(),
				CreatedOn:            whitelist.CreatedOn,
				UpdatedOn:            whitelist.UpdatedOn,
			})
		}

		ctx.JSON(http.StatusOK, wlExported)
	}
}

func (b *blocklistHandler) onAPIDeleteWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		errSave := b.blocklists.DeleteSteamBlockWhitelists(ctx, steamID)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (b *blocklistHandler) onAPIGetBlockListSources() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		blockLists, err := b.blocklists.GetCIDRBlockSources(ctx)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, blockLists)
	}
}

func (b *blocklistHandler) onAPIPostBlockListCreate() gin.HandlerFunc {
	type createRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		blockList, errSave := b.blocklists.CreateCIDRBlockSources(ctx, req.Name, req.URL, req.Enabled)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Could not create CIDR block source."))

			return
		}

		ctx.JSON(http.StatusCreated, blockList)
	}
}

func (b *blocklistHandler) onAPIPostBlockListUpdate() gin.HandlerFunc {
	type updateRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		sourceID, idFound := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if !idFound {
			return
		}

		var req updateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		blockSource, errUpdate := b.blocklists.UpdateCIDRBlockSource(ctx, sourceID, req.Name, req.URL, req.Enabled)
		if errUpdate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errUpdate, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, blockSource)
	}
}

func (b *blocklistHandler) onAPICreateWhitelistIP() gin.HandlerFunc {
	type createRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		whitelist, errSave := b.blocklists.CreateCIDRBlockWhitelist(ctx, req.Address)
		if errSave != nil {
			if errors.Is(errSave, ErrInvalidCIDR) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest, "CIDR invalid"))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusCreated, CIDRBlockWhitelistExport{
			CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
			Address:              whitelist.Address.String(),
			CreatedOn:            whitelist.CreatedOn,
			UpdatedOn:            whitelist.UpdatedOn,
		})
	}
}

func (b *blocklistHandler) onAPIUpdateWhitelistIP() gin.HandlerFunc {
	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, idFound := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if !idFound {
			return
		}

		var req updateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if !strings.Contains(req.Address, "/") {
			req.Address += "/32"
		}

		whiteList, errSave := b.blocklists.UpdateCIDRBlockWhitelist(ctx, whitelistID, req.Address)
		if errSave != nil {
			if errors.Is(errSave, ErrInvalidCIDR) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest, "CIDR invalid"))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, whiteList)
	}
}

func (b *blocklistHandler) onAPIDeleteBlockListWhitelist() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whitelistID, idFound := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if !idFound {
			return
		}

		if err := b.blocklists.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (b *blocklistHandler) onAPIPostBlocklistCheck() gin.HandlerFunc {
	type checkReq struct {
		Address netip.Addr `json:"address"`
	}

	type checkResp struct {
		Blocked bool   `json:"blocked"`
		Source  string `json:"source"`
	}

	return func(ctx *gin.Context) {
		var req checkReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ctx.JSON(http.StatusInternalServerError, checkResp{Blocked: false, Source: ""})
	}
}
