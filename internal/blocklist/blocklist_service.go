package blocklist

import (
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type blocklistHandler struct {
	blocklists domain.BlocklistUsecase
	networks   domain.NetworkUsecase
}

func NewBlocklistHandler(engine *gin.Engine, bu domain.BlocklistUsecase, nu domain.NetworkUsecase, ath domain.AuthUsecase) {
	handler := blocklistHandler{
		blocklists: bu,
		networks:   nu,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
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
		admin := adminGrp.Use(ath.AuthMiddleware(domain.PAdmin))
		admin.POST("/api/block_list/sources", handler.onAPIPostBlockListCreate())
		admin.POST("/api/block_list/sources/:cidr_block_source_id", handler.onAPIPostBlockListUpdate())
		admin.DELETE("/api/block_list/sources/:cidr_block_source_id", handler.onAPIDeleteBlockList())
	}
}

type (
	CIDRBlockWhitelistExport struct {
		CIDRBlockWhitelistID int    `json:"cidr_block_whitelist_id"`
		Address              string `json:"address"`
		domain.TimeStamped
	}
)

func (b *blocklistHandler) onAPIWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.blocklists.GetSteamBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load ip whitelist", log.ErrAttr(errWl))

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
			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrInvalidSID)
			slog.Warn("Got invalid steamid", slog.String("steam_id", req.SteamIDValue))

			return
		}

		whitelist, errSave := b.blocklists.CreateSteamBlockWhitelists(ctx, steamID)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save steam block whitelist", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, whitelist)
	}
}

func (b *blocklistHandler) onAPIDeleteBlockList() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sourceID, errSourceID := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if errSourceID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to parse cidr_block_source_id query")

			return
		}

		if err := b.blocklists.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete blocklist", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (b *blocklistHandler) onAPIWhitelistIPs() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whiteLists, errWl := b.blocklists.GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load ip whitelist", log.ErrAttr(errWl))

			return
		}

		var wlExported []CIDRBlockWhitelistExport
		for _, whitelist := range whiteLists {
			wlExported = append(wlExported, CIDRBlockWhitelistExport{
				CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
				Address:              whitelist.Address.String(),
				TimeStamped:          whitelist.TimeStamped,
			})
		}

		ctx.JSON(http.StatusOK, wlExported)
	}
}

func (b *blocklistHandler) onAPIDeleteWhitelistSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := httphelper.GetSID64Param(ctx, "steam_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to delete steam whitelist", slog.String("steam_id", steamID.String()))

			return
		}

		errSave := b.blocklists.DeleteSteamBlockWhitelists(ctx, steamID)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save whitelist", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (b *blocklistHandler) onAPIGetBlockListSources() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		blockLists, err := b.blocklists.GetCIDRBlockSources(ctx)
		if err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load blocklist sources", log.ErrAttr(err))

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
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save blocklist", log.ErrAttr(errSave))

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
		sourceID, err := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if err != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid cidr_block_source_id", slog.Int("cidr_block_source_id", sourceID))

			return
		}

		var req updateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		blockSource, errUpdate := b.blocklists.UpdateCIDRBlockSource(ctx, sourceID, req.Name, req.URL, req.Enabled)
		if errUpdate != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to create ip whitelist", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, CIDRBlockWhitelistExport{
			CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
			Address:              whitelist.Address.String(),
			TimeStamped:          whitelist.TimeStamped,
		})
	}
}

func (b *blocklistHandler) onAPIUpdateWhitelistIP() gin.HandlerFunc {
	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, errID := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Git invalid cidr_block_whitelist_id")

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
			httphelper.HandleErrInvalidFormat(ctx)
			slog.Error("Failed to save whitelist", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, whiteList)
	}
}

func (b *blocklistHandler) onAPIDeleteBlockListWhitelist() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		whitelistID, errWhitelistID := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errWhitelistID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Git invalid cidr_block_whitelist_id")

			return
		}

		if err := b.blocklists.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete whitelist", log.ErrAttr(err))

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
