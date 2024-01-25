package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"sort"
)

type ServersHandler struct {
	ServersUsecase domain.ServersUsecase
	log            *zap.Logger
}

func NewServerHandler(logger *zap.Logger, engine *gin.Engine, serversUsecase domain.ServersUsecase) {
	handler := &ServersHandler{
		ServersUsecase: serversUsecase,
		log:            logger,
	}

	engine.GET("/api/servers/state", handler.onAPIGetServerStates())
	engine.GET("/api/servers", handler.onAPIGetServers())

	// admin
	//adminRoute := adminGrp.Use(authMiddleware(env, domain.PAdmin))
	engine.POST("/api/servers", handler.onAPIPostServer())
	engine.POST("/api/servers/:server_id", handler.onAPIPostServerUpdate())
	engine.DELETE("/api/servers/:server_id", handler.onAPIPostServerDelete())
	engine.POST("/api/servers_admin", handler.onAPIGetServersAdmin())

}

type serverInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

func (s *ServersHandler) onAPIGetServers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := s.ServersUsecase.GetServers(ctx, domain.ServerQueryFilter{})
		if errServers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

func (s *ServersHandler) onAPIGetServerStates() gin.HandlerFunc {
	type UserServers struct {
		Servers []domain.BaseServer `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	return func(ctx *gin.Context) {
		var (
			lat = http_helper.GetDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = http_helper.GetDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = env.State().Current()
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

func (s *ServersHandler) onAPIPostServer() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !http_helper.Bind(ctx, log, &req) {
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

		if errSave := s.ServersUsecase.SaveServer(ctx, &server); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
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

func (s *ServersHandler) onAPIPostServerUpdate() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := http_helper.GetIntParam(ctx, "server_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var server domain.Server
		if errServer := s.ServersUsecase.GetServer(ctx, serverID, &server); errServer != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req serverUpdateRequest
		if !http_helper.Bind(ctx, log, &req) {
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

		if errSave := s.ServersUsecase.SaveServer(ctx, &server); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("ServerStore config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func (s *ServersHandler) onAPIGetServersAdmin() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter domain.ServerQueryFilter
		if !http_helper.Bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := s.ServersUsecase.GetServers(ctx, filter)
		if errServers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, servers))
	}
}

func (s *ServersHandler) onAPIPostServerDelete() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := http_helper.GetIntParam(ctx, "server_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var server domain.Server
		if errServer := s.ServersUsecase.GetServer(ctx, serverID, &server); errServer != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		server.Deleted = true

		if errSave := s.ServersUsecase.SaveServer(ctx, &server); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("ServerStore config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}
