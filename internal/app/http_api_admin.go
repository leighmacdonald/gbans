package app

import (
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func onAPIPostServer(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server := store.NewServer(req.ServerNameShort, req.Host, req.Port)
		server.Name = req.ServerName
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save new server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config created",
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

func onAPIPostServerUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostServerDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		server.Deleted = true

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("Server config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIGetServersAdmin(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter store.ServerQueryFilter
		if !bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := app.db.GetServers(ctx, filter)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, servers))
	}
}

func onAPIPutPlayerPermission(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updatePermissionLevel struct {
		PermissionLevel consts.Privilege `json:"permission_level"`
	}

	return func(ctx *gin.Context) {
		steamID, errParam := getSID64Param(ctx, "steam_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req updatePermissionLevel
		if !bind(ctx, log, &req) {
			return
		}

		var person store.Person
		if errGet := app.db.GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == app.conf.General.Owner {
			responseErr(ctx, http.StatusConflict, errors.New("Cannot alter site owner permissions"))

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := app.db.SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}

func onAPIDeleteBlockList(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sourceID, errSourceID := getIntParam(ctx, "cidr_block_source_id")
		if errSourceID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if err := app.db.DeleteCIDRBlockSources(ctx, sourceID); err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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
	store.TimeStamped
}

func onAPIGetBlockLists(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type BlockSources struct {
		Sources   []store.CIDRBlockSource    `json:"sources"`
		Whitelist []CIDRBlockWhitelistExport `json:"whitelist"`
	}

	return func(ctx *gin.Context) {
		blockLists, err := app.db.GetCIDRBlockSources(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to load blocklist", zap.Error(err))

			return
		}

		whiteLists, errWl := app.db.GetCIDRBlockWhitelists(ctx)
		if errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIPostBlockListCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		parsedURL, errURL := url.Parse(req.URL)
		if errURL != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		blockList := store.CIDRBlockSource{
			Name:        req.Name,
			URL:         parsedURL.String(),
			Enabled:     req.Enabled,
			TimeStamped: store.NewTimeStamped(),
		}

		if errSave := app.db.SaveCIDRBlockSources(ctx, &blockList); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save blocklist", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, blockList)
	}
}

func onAPIPostBlockListUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled bool   `json:"enabled"`
	}

	return func(ctx *gin.Context) {
		sourceID, err := getIntParam(ctx, "cidr_block_source_id")
		if err != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var blockSource store.CIDRBlockSource

		if errSource := app.db.GetCIDRBlockSource(ctx, sourceID, &blockSource); errSource != nil {
			if errors.Is(errSource, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			}

			return
		}

		var req updateRequest
		if !bind(ctx, log, &req) {
			return
		}

		testBlocker := NewNetworkBlocker()
		if count, errTest := testBlocker.AddRemoteSource(ctx, req.Name, req.URL); errTest != nil || count == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			if errTest != nil {
				log.Error("Failed to validate blocklist url", zap.Error(errTest))
			} else {
				log.Error("Blocklist returned no valid results")
			}

			return
		}

		blockSource.Enabled = req.Enabled
		blockSource.Name = req.Name
		blockSource.URL = req.URL

		if errUpdate := app.db.SaveCIDRBlockSources(ctx, &blockSource); errUpdate != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, blockSource)
	}
}

func onAPIPostBlockListWhitelistCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		whitelist := store.CIDRBlockWhitelist{
			Address:     cidr,
			TimeStamped: store.NewTimeStamped(),
		}

		if errSave := app.db.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, CIDRBlockWhitelistExport{
			CIDRBlockWhitelistID: whitelist.CIDRBlockWhitelistID,
			Address:              whitelist.Address.String(),
			TimeStamped:          whitelist.TimeStamped,
		})

		app.netBlock.AddWhitelist(whitelist.CIDRBlockWhitelistID, cidr)
	}
}

func onAPIPostBlockListWhitelistUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateRequest struct {
		Address string `json:"address"`
	}

	return func(ctx *gin.Context) {
		whitelistID, errID := getIntParam(ctx, "cidr_block_whitelist_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req updateRequest
		if !bind(ctx, log, &req) {
			return
		}

		_, cidr, errParse := net.ParseCIDR(req.Address)
		if errParse != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var whitelist store.CIDRBlockWhitelist
		if errGet := app.db.GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		whitelist.Address = cidr

		if errSave := app.db.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save whitelist", zap.Error(errSave))

			return
		}
	}
}

func onAPIDeleteBlockListWhitelist(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		whitelistID, errWhitelistID := getIntParam(ctx, "cidr_block_whitelist_id")
		if errWhitelistID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if err := app.db.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to delete whitelist", zap.Error(err))

			return
		}

		log.Info("Blocklist deleted", zap.Int("cidr_block_source_id", whitelistID))

		ctx.JSON(http.StatusOK, nil)

		app.netBlock.RemoveWhitelist(whitelistID)
	}
}

func onAPIPostBlocklistCheck(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		isBlocked, source := app.netBlock.IsMatch(ipAddr)

		ctx.JSON(http.StatusOK, checkResp{
			Blocked: isBlocked,
			Source:  source,
		})
	}
}
