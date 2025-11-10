package network

import (
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type blocklistHandler struct {
	Blocklists

	networks Networks
}

func NewBlocklistHandler(engine *gin.Engine, ath httphelper.Authenticator, blocklist Blocklists, networks Networks) {
	handler := blocklistHandler{Blocklists: blocklist, networks: networks}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.Middleware(permission.Moderator))
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
		admin := adminGrp.Use(ath.Middleware(permission.Admin))
		admin.POST("/api/block_list/sources", handler.onAPIPostBlockListCreate())
		admin.POST("/api/block_list/sources/:cidr_block_source_id", handler.onAPIPostBlockListUpdate())
		admin.DELETE("/api/block_list/sources/:cidr_block_source_id", handler.onAPIDeleteBlockList())
	}
}

type CIDRBlockWhitelistExport struct {
	CIDRBlockWhitelistID int       `json:"cidr_block_whitelist_id"`
	Address              string    `json:"address"`
	CreatedOn            time.Time `json:"created_on"`
	UpdatedOn            time.Time `json:"updated_on"`
}

func (b *blocklistHandler) onAPIWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.GetSteamBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errWl, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, whiteLists)
	}
}

type CreateSteamWhitelistRequest struct {
	httphelper.SteamIDField
}

func (b *blocklistHandler) onAPICreateWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateSteamWhitelistRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		steamID, ok := req.SteamID(ctx)
		if !ok {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, steamid.ErrInvalidSID))

			return
		}

		whitelist, errSave := b.CreateSteamBlockWhitelists(ctx, steamID)
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

		if err := b.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal),
				"Could not delete blocklist source: %d", sourceID))

			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (b *blocklistHandler) onAPIWhitelistIPs() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.GetCIDRBlockWhitelists(ctx)
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

		errSave := b.DeleteSteamBlockWhitelists(ctx, steamID)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (b *blocklistHandler) onAPIGetBlockListSources() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		blockLists, err := b.GetCIDRBlockSources(ctx)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, blockLists)
	}
}

type BlocklistCreateRequest struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func (b *blocklistHandler) onAPIPostBlockListCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req BlocklistCreateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		blockList, errSave := b.CreateCIDRBlockSources(ctx, req.Name, req.URL, req.Enabled)
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

		blockSource, errUpdate := b.UpdateCIDRBlockSource(ctx, sourceID, req.Name, req.URL, req.Enabled)
		if errUpdate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errUpdate, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, blockSource)
	}
}

type CreateWhitelistIPRequest struct {
	Address string `json:"address"`
}

func (b *blocklistHandler) onAPICreateWhitelistIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateWhitelistIPRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		whitelist, errSave := b.CreateCIDRBlockWhitelist(ctx, req.Address)
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

type UpdateWhitelistIPRequest struct {
	Address string `json:"address"`
}

func (b *blocklistHandler) onAPIUpdateWhitelistIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whitelistID, idFound := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if !idFound {
			return
		}

		var req UpdateWhitelistIPRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if !strings.Contains(req.Address, "/") {
			req.Address += maskSingleHost
		}

		whiteList, errSave := b.UpdateCIDRBlockWhitelist(ctx, whitelistID, req.Address)
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

		if err := b.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
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
