package blocklist

import (
	"net"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type BlocklistHandler struct {
	BlocklistUsecase domain.BlocklistUsecase
	nu               domain.NetworkUsecase
	log              *zap.Logger
}

func NewBlocklistHandler(log *zap.Logger, engine *gin.Engine, bu domain.BlocklistUsecase, nu domain.NetworkUsecase, ath domain.AuthUsecase) {
	handler := BlocklistHandler{
		BlocklistUsecase: bu,
		nu:               nu,
		log:              log,
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/block_list/whitelist", handler.onAPIPostBlockListWhitelistCreate())
		mod.POST("/api/block_list/whitelist/:cidr_block_whitelist_id", handler.onAPIPostBlockListWhitelistUpdate())
		mod.DELETE("/api/block_list/whitelist/:cidr_block_whitelist_id", handler.onAPIDeleteBlockListWhitelist())
		mod.GET("/api/block_list", handler.onAPIGetBlockLists())
		mod.POST("/api/block_list/checker", handler.onAPIPostBlocklistCheck())
	}

	// admin
	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(ath.AuthMiddleware(domain.PAdmin))
		admin.POST("/api/block_list", handler.onAPIPostBlockListCreate())
		admin.POST("/api/block_list/:cidr_block_source_id", handler.onAPIPostBlockListUpdate())
		admin.DELETE("/api/block_list/:cidr_block_source_id", handler.onAPIDeleteBlockList())
	}
}

type (
	CIDRBlockWhitelistExport struct {
		CIDRBlockWhitelistID int    `json:"cidr_block_whitelist_id"`
		Address              string `json:"address"`
		domain.TimeStamped
	}
)

func (b *BlocklistHandler) onAPIDeleteBlockList() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sourceID, errSourceID := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if errSourceID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := b.BlocklistUsecase.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to delete blocklist", zap.Error(err))

			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (b *BlocklistHandler) onAPIGetBlockLists() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type BlockSources struct {
		Sources   []domain.CIDRBlockSource   `json:"sources"`
		Whitelist []CIDRBlockWhitelistExport `json:"whitelist"`
	}

	return func(ctx *gin.Context) {
		blockLists, err := b.BlocklistUsecase.GetCIDRBlockSources(ctx)
		if err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to load blocklist", zap.Error(err))

			return
		}

		whiteLists, errWl := b.BlocklistUsecase.GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to load blocklist", zap.Error(err))

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

		ctx.JSON(http.StatusOK, BlockSources{Sources: blockLists, Whitelist: wlExported})
	}
}

func (b *BlocklistHandler) onAPIPostBlockListCreate() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type createRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		blockList, errSave := b.BlocklistUsecase.CreateCIDRBlockSources(ctx, req.Name, req.URL, req.Enabled)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to save blocklist", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, blockList)
	}
}

func (b *BlocklistHandler) onAPIPostBlockListUpdate() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		sourceID, err := httphelper.GetIntParam(ctx, "cidr_block_source_id")
		if err != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req updateRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		blockSource, errUpdate := b.BlocklistUsecase.UpdateCIDRBlockSource(ctx, sourceID, req.Name, req.URL, req.Enabled)
		if errUpdate != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, blockSource)
	}
}

func (b *BlocklistHandler) onAPIPostBlockListWhitelistCreate() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type createRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		whitelist, errSave := b.BlocklistUsecase.CreateCIDRBlockWhitelist(ctx, req.Address)
		if errSave != nil {
			_ = httphelper.ErrorHandledWithReturn(ctx, errSave)

			return
		}

		ctx.JSON(http.StatusCreated, CIDRBlockWhitelistExport{
			CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
			Address:              whitelist.Address.String(),
			TimeStamped:          whitelist.TimeStamped,
		})

		b.nu.AddWhitelist(whitelist.CIDRBlockWhitelistID, whitelist.Address)
	}
}

func (b *BlocklistHandler) onAPIPostBlockListWhitelistUpdate() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, errID := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req updateRequest
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		whiteList, errSave := b.BlocklistUsecase.UpdateCIDRBlockWhitelist(ctx, whitelistID, req.Address)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to save whitelist", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, whiteList)
	}
}

func (b *BlocklistHandler) onAPIDeleteBlockListWhitelist() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		whitelistID, errWhitelistID := httphelper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errWhitelistID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := b.BlocklistUsecase.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to delete whitelist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", whitelistID))

		ctx.JSON(http.StatusOK, nil)

		b.nu.RemoveWhitelist(whitelistID)
	}
}

func (b *BlocklistHandler) onAPIPostBlocklistCheck() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type checkReq struct {
		Address string `json:"address"`
	}

	type checkResp struct {
		Blocked bool   `json:"blocked"`
		Source  string `json:"source"`
	}

	return func(ctx *gin.Context) {
		var req checkReq
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		ipAddr := net.ParseIP(req.Address)
		if ipAddr == nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		source, isBlocked := b.nu.IsMatch(ipAddr)

		ctx.JSON(http.StatusOK, checkResp{
			Blocked: isBlocked,
			Source:  source,
		})
	}
}
