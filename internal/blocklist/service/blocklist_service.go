package service

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
)

type BlocklistHandler struct {
	BlocklistUsecase domain.BlocklistUsecase
	log              *zap.Logger
}

func NewBlocklistHandler(engine *gin.Engine, log *zap.Logger, bu domain.BlocklistUsecase) {
	handler := BlocklistHandler{
		BlocklistUsecase: bu,
		log:              log,
	}

	// mod
	engine.POST("/api/block_list/whitelist", handler.onAPIPostBlockListWhitelistCreate())
	engine.POST("/api/block_list/whitelist/:cidr_block_whitelist_id", handler.onAPIPostBlockListWhitelistUpdate())
	engine.DELETE("/api/block_list/whitelist/:cidr_block_whitelist_id", handler.onAPIDeleteBlockListWhitelist())
	engine.GET("/api/block_list", handler.onAPIGetBlockLists())
	engine.POST("/api/block_list/checker", handler.onAPIPostBlocklistCheck())

	// admin
	engine.POST("/api/block_list", handler.onAPIPostBlockListCreate())
	engine.POST("/api/block_list/:cidr_block_source_id", handler.onAPIPostBlockListUpdate())
	engine.DELETE("/api/block_list/:cidr_block_source_id", handler.onAPIDeleteBlockList())
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
		sourceID, errSourceID := http_helper.GetIntParam(ctx, "cidr_block_source_id")
		if errSourceID != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		if err := b.BlocklistUsecase.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			log.Error("Failed to delete blocklist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", sourceID))

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
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			log.Error("Failed to load blocklist", zap.Error(err))

			return
		}

		whiteLists, errWl := b.BlocklistUsecase.GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

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
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Name == "" {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		parsedURL, errURL := url.Parse(req.URL)
		if errURL != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		blockList := domain.CIDRBlockSource{
			Name:        req.Name,
			URL:         parsedURL.String(),
			Enabled:     req.Enabled,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := b.BlocklistUsecase.SaveCIDRBlockSources(ctx, &blockList); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

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
		sourceID, err := http_helper.GetIntParam(ctx, "cidr_block_source_id")
		if err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		var blockSource domain.CIDRBlockSource

		if errSource := b.BlocklistUsecase.GetCIDRBlockSource(ctx, sourceID, &blockSource); errSource != nil {
			if errors.Is(errSource, errs.ErrNoResult) {
				http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)
			}

			return
		}

		var req updateRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		// testBlocker := network.NewBlocker()
		// if count, errTest := testBlocker.AddRemoteSource(ctx, req.Name, req.URL); errTest != nil || count == 0 {
		//	http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
		//
		//	if errTest != nil {
		//		log.Error("Failed to validate blocklist url", zap.Error(errTest))
		//	} else {
		//		log.Error("Blocklist returned no valid results")
		//	}
		//
		//	return
		// }

		blockSource.Enabled = req.Enabled
		blockSource.Name = req.Name
		blockSource.URL = req.URL

		if errUpdate := b.BlocklistUsecase.SaveCIDRBlockSources(ctx, &blockSource); errUpdate != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

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
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if !strings.Contains(req.Address, "/") {
			req.Address += "/32"
		}

		_, cidr, errParse := net.ParseCIDR(req.Address)
		if errParse != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		whitelist := domain.CIDRBlockWhitelist{
			Address:     cidr,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := b.BlocklistUsecase.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, CIDRBlockWhitelistExport{
			CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
			Address:              whitelist.Address.String(),
			TimeStamped:          whitelist.TimeStamped,
		})

		env.NetBlocks().AddWhitelist(whitelist.CIDRBlockWhitelistID, cidr)
	}
}

func (b *BlocklistHandler) onAPIPostBlockListWhitelistUpdate() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, errID := http_helper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errID != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		var req updateRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		_, cidr, errParse := net.ParseCIDR(req.Address)
		if errParse != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		var whitelist domain.CIDRBlockWhitelist
		if errGet := b.BlocklistUsecase.GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		whitelist.Address = cidr

		if errSave := b.BlocklistUsecase.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			log.Error("Failed to save whitelist", zap.Error(errSave))

			return
		}
	}
}

func (b *BlocklistHandler) onAPIDeleteBlockListWhitelist() gin.HandlerFunc {
	log := b.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		whitelistID, errWhitelistID := http_helper.GetIntParam(ctx, "cidr_block_whitelist_id")
		if errWhitelistID != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		if err := b.BlocklistUsecase.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.domain.ErrInternal)

			log.Error("Failed to delete whitelist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", whitelistID))

		ctx.JSON(http.StatusOK, nil)

		env.NetBlocks().RemoveWhitelist(whitelistID)
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
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		ipAddr := net.ParseIP(req.Address)
		if ipAddr == nil {
			http_helper.http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.domain.ErrBadRequest)

			return
		}

		isBlocked, source := env.NetBlocks().IsMatch(ipAddr)

		ctx.JSON(http.StatusOK, checkResp{
			Blocked: isBlocked,
			Source:  source,
		})
	}
}
