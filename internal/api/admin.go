package api

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
	"go.uber.org/zap"
)

func onAPIPostServer(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server := domain.NewServer(req.ServerNameShort, req.Host, req.Port)
		server.Name = req.ServerName
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := env.Store().SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save new server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("ServerStore config created",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

type serverUpdateRequest struct {
	ServerName      string  `json:"server_name"`
	ServerNameShort string  `json:"server_name_short"`
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	ReservedSlots   int     `json:"reserved_slots"`
	RCON            string  `json:"rcon"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	CC              string  `json:"cc"`
	DefaultMap      string  `json:"default_map"`
	Region          string  `json:"region"`
	IsEnabled       bool    `json:"is_enabled"`
	EnableStats     bool    `json:"enable_stats"`
	LogSecret       int     `json:"log_secret"`
}

func onAPIPostServerUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var server domain.Server
		if errServer := env.Store().GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server.ShortName = req.ServerNameShort
		server.Name = req.ServerName
		server.Address = req.Host
		server.Port = req.Port
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled
		server.LogSecret = req.LogSecret
		server.EnableStats = req.EnableStats

		if errSave := env.Store().SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("ServerStore config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostServerDelete(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var server domain.Server
		if errServer := env.Store().GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		server.Deleted = true

		if errSave := env.Store().SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("ServerStore config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIGetServersAdmin(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter domain.ServerQueryFilter
		if !bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := env.Store().GetServers(ctx, filter)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, servers))
	}
}

func onAPIPutPlayerPermission(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updatePermissionLevel struct {
		PermissionLevel domain.Privilege `json:"permission_level"`
	}

	return func(ctx *gin.Context) {
		steamID, errParam := getSID64Param(ctx, "steam_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var req updatePermissionLevel
		if !bind(ctx, log, &req) {
			return
		}

		var person domain.Person
		if errGet := env.Store().GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == env.Config().General.Owner {
			responseErr(ctx, http.StatusConflict, errPermissionDenied)

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}

func onAPIDeleteBlockList(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sourceID, errSourceID := getIntParam(ctx, "cidr_block_source_id")
		if errSourceID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if err := env.Store().DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to delete blocklist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", sourceID))

		ctx.JSON(http.StatusOK, nil)
	}
}

type CIDRBlockWhitelistExport struct {
	CIDRBlockWhitelistID int    `json:"cidr_block_whitelist_id"`
	Address              string `json:"address"`
	domain.TimeStamped
}

func onAPIGetBlockLists(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type BlockSources struct {
		Sources   []domain.CIDRBlockSource   `json:"sources"`
		Whitelist []CIDRBlockWhitelistExport `json:"whitelist"`
	}

	return func(ctx *gin.Context) {
		blockLists, err := env.Store().GetCIDRBlockSources(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to load blocklist", zap.Error(err))

			return
		}

		whiteLists, errWl := env.Store().GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

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

func onAPIPostBlockListCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type createRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !bind(ctx, log, &req) {
			return
		}

		if req.Name == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		parsedURL, errURL := url.Parse(req.URL)
		if errURL != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		blockList := domain.CIDRBlockSource{
			Name:        req.Name,
			URL:         parsedURL.String(),
			Enabled:     req.Enabled,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := env.Store().SaveCIDRBlockSources(ctx, &blockList); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save blocklist", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, blockList)
	}
}

func onAPIPostBlockListUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		sourceID, err := getIntParam(ctx, "cidr_block_source_id")
		if err != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var blockSource domain.CIDRBlockSource

		if errSource := env.Store().GetCIDRBlockSource(ctx, sourceID, &blockSource); errSource != nil {
			if errors.Is(errSource, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusBadRequest, errBadRequest)
			}

			return
		}

		var req updateRequest
		if !bind(ctx, log, &req) {
			return
		}

		// testBlocker := network.NewBlocker()
		// if count, errTest := testBlocker.AddRemoteSource(ctx, req.Name, req.URL); errTest != nil || count == 0 {
		//	responseErr(ctx, http.StatusBadRequest, errBadRequest)
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

		if errUpdate := env.Store().SaveCIDRBlockSources(ctx, &blockSource); errUpdate != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, blockSource)
	}
}

func onAPIPostBlockListWhitelistCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type createRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		var req createRequest
		if !bind(ctx, log, &req) {
			return
		}

		if !strings.Contains(req.Address, "/") {
			req.Address += "/32"
		}

		_, cidr, errParse := net.ParseCIDR(req.Address)
		if errParse != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		whitelist := domain.CIDRBlockWhitelist{
			Address:     cidr,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := env.Store().SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

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

func onAPIPostBlockListWhitelistUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, errID := getIntParam(ctx, "cidr_block_whitelist_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var req updateRequest
		if !bind(ctx, log, &req) {
			return
		}

		_, cidr, errParse := net.ParseCIDR(req.Address)
		if errParse != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var whitelist domain.CIDRBlockWhitelist
		if errGet := env.Store().GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		whitelist.Address = cidr

		if errSave := env.Store().SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save whitelist", zap.Error(errSave))

			return
		}
	}
}

func onAPIDeleteBlockListWhitelist(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		whitelistID, errWhitelistID := getIntParam(ctx, "cidr_block_whitelist_id")
		if errWhitelistID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if err := env.Store().DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to delete whitelist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", whitelistID))

		ctx.JSON(http.StatusOK, nil)

		env.NetBlocks().RemoveWhitelist(whitelistID)
	}
}

func onAPIPostBlocklistCheck(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type checkReq struct {
		Address string `json:"address"`
	}

	type checkResp struct {
		Blocked bool   `json:"blocked"`
		Source  string `json:"source"`
	}

	return func(ctx *gin.Context) {
		var req checkReq
		if !bind(ctx, log, &req) {
			return
		}

		ipAddr := net.ParseIP(req.Address)
		if ipAddr == nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		isBlocked, source := env.NetBlocks().IsMatch(ipAddr)

		ctx.JSON(http.StatusOK, checkResp{
			Blocked: isBlocked,
			Source:  source,
		})
	}
}
