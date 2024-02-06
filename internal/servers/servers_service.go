package servers

import (
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type serversHandler struct {
	serversUsecase domain.ServersUsecase
	stateUsecase   domain.StateUsecase
	pu             domain.PersonUsecase
	log            *zap.Logger
}

func NewServerHandler(logger *zap.Logger, engine *gin.Engine, serversUsecase domain.ServersUsecase,
	stateUsecase domain.StateUsecase, ath domain.AuthUsecase,
) {
	handler := &serversHandler{
		serversUsecase: serversUsecase,
		stateUsecase:   stateUsecase,
		log:            logger,
	}

	engine.GET("/export/sourcemod/admins_simple.ini", handler.onAPIExportSourcemodSimpleAdmins())
	engine.GET("/api/servers/state", handler.onAPIGetServerStates())
	engine.GET("/api/servers", handler.onAPIGetServers())

	// admin
	srvGrp := engine.Group("/")
	{
		admin := srvGrp.Use(ath.AuthMiddleware(domain.PAdmin))
		admin.POST("/api/servers", handler.onAPIPostServer())
		admin.POST("/api/servers/:server_id", handler.onAPIPostServerUpdate())
		admin.DELETE("/api/servers/:server_id", handler.onAPIPostServerDelete())
		admin.POST("/api/servers_admin", handler.onAPIGetServersAdmin())
	}
}

type serverInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

func (h *serversHandler) onAPIExportSourcemodSimpleAdmins() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := h.pu.GetSteamIdsAbove(ctx, domain.PReserved)
		if errPrivilegedIds != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		players, errPlayers := h.pu.GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})

		bld := strings.Builder{}

		for _, player := range players {
			var perms string

			switch player.PermissionLevel {
			case domain.PAdmin:
				perms = "z"
			case domain.PModerator:
				perms = "abcdefgjk"
			case domain.PEditor:
				perms = "ak"
			case domain.PReserved:
				perms = "a"
			}

			if perms == "" {
				log.Warn("User has no perm string", zap.Int64("sid", player.SteamID.Int64()))
			} else {
				bld.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", steamid.SID64ToSID3(player.SteamID), perms))
			}
		}

		ctx.String(http.StatusOK, bld.String())
	}
}

func (h *serversHandler) onAPIGetServers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := h.serversUsecase.GetServers(ctx, domain.ServerQueryFilter{})
		if errServers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var servers []serverInfoSafe
		for _, server := range fullServers {
			servers = append(servers, serverInfoSafe{
				ServerNameLong: server.Name,
				ServerName:     server.ShortName,
				ServerID:       server.ServerID,
				Colour:         "",
			})
		}

		ctx.JSON(http.StatusOK, servers)
	}
}

func (h *serversHandler) onAPIGetServerStates() gin.HandlerFunc {
	type UserServers struct {
		Servers []domain.BaseServer `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	distance := func(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
		radianLat1 := math.Pi * lat1 / 180
		radianLat2 := math.Pi * lat2 / 180
		theta := lng1 - lng2
		radianTheta := math.Pi * theta / 180

		dist := math.Sin(radianLat1)*math.Sin(radianLat2) + math.Cos(radianLat1)*math.Cos(radianLat2)*math.Cos(radianTheta)
		if dist > 1 {
			dist = 1
		}

		dist = math.Acos(dist)
		dist = dist * 180 / math.Pi
		dist = dist * 60 * 1.1515
		dist *= 1.609344 // convert to km

		return dist
	}

	return func(ctx *gin.Context) {
		var (
			lat = httphelper.GetDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = httphelper.GetDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = h.stateUsecase.Current()
			servers  []domain.BaseServer
		)

		for _, srv := range curState {
			servers = append(servers, domain.BaseServer{
				Host:       srv.Host,
				Port:       srv.Port,
				IP:         srv.IP,
				Name:       srv.Name,
				NameShort:  srv.NameShort,
				Region:     srv.Region,
				CC:         srv.CC,
				ServerID:   srv.ServerID,
				Players:    srv.PlayerCount,
				MaxPlayers: srv.MaxPlayers,
				Bots:       srv.Bots,
				Map:        srv.Map,
				GameTypes:  []string{},
				Latitude:   srv.Latitude,
				Longitude:  srv.Longitude,
				Distance:   distance(srv.Latitude, srv.Longitude, lat, lon),
			})
		}

		sort.SliceStable(servers, func(i, j int) bool {
			return servers[i].Name < servers[j].Name
		})

		ctx.JSON(http.StatusOK, UserServers{
			Servers: servers,
			LatLong: ip2location.LatLong{
				Latitude:  lat,
				Longitude: lon,
			},
		})
	}
}

func (h *serversHandler) onAPIPostServer() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !httphelper.Bind(ctx, log, &req) {
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

		if errSave := h.serversUsecase.SaveServer(ctx, &server); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
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

func (h *serversHandler) onAPIPostServerUpdate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := httphelper.GetIntParam(ctx, "server_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		server, errServer := h.serversUsecase.GetServer(ctx, serverID)
		if errServer != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req serverUpdateRequest
		if !httphelper.Bind(ctx, log, &req) {
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

		if errSave := h.serversUsecase.SaveServer(ctx, &server); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("ServerStore config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func (h *serversHandler) onAPIGetServersAdmin() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter domain.ServerQueryFilter
		if !httphelper.Bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := h.serversUsecase.GetServers(ctx, filter)
		if errServers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, servers))
	}
}

func (h *serversHandler) onAPIPostServerDelete() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := httphelper.GetIntParam(ctx, "server_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		server, errServer := h.serversUsecase.GetServer(ctx, serverID)
		if errServer != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		server.Deleted = true

		if errSave := h.serversUsecase.SaveServer(ctx, &server); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("ServerStore config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}
