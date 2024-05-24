package servers

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type serversHandler struct {
	serversUsecase domain.ServersUsecase
	stateUsecase   domain.StateUsecase
	personUsecase  domain.PersonUsecase
}

func NewServerHandler(engine *gin.Engine, serversUsecase domain.ServersUsecase,
	stateUsecase domain.StateUsecase, ath domain.AuthUsecase, personUsecase domain.PersonUsecase,
) {
	handler := &serversHandler{
		serversUsecase: serversUsecase,
		stateUsecase:   stateUsecase,
		personUsecase:  personUsecase,
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
	return func(ctx *gin.Context) {
		privilegedIDs, errPrivilegedIDs := h.personUsecase.GetSteamIDsAbove(ctx, domain.PReserved)
		if errPrivilegedIDs != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		players, errPlayers := h.personUsecase.GetPeopleBySteamID(ctx, privilegedIDs)
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
				slog.Warn("User has no perm string", slog.Int64("sid", player.SteamID.Int64()))
			} else {
				bld.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", player.SteamID.Steam3(), perms))
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
		Servers []domain.SafeServer `json:"servers"`
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
			servers  []domain.SafeServer
		)

		for _, srv := range curState {
			servers = append(servers, domain.SafeServer{
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
	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		server := domain.NewServer(req.ServerNameShort, req.Host, req.Port)
		server.Name = req.ServerName
		server.Password = req.Password
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled
		server.LogSecret = req.LogSecret

		if errSave := h.serversUsecase.SaveServer(ctx, &server); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save new server", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		slog.Info("ServerStore config created",
			slog.Int("server_id", server.ServerID),
			slog.String("name", server.ShortName))
	}
}

type serverUpdateRequest struct {
	ServerName      string  `json:"server_name"`
	ServerNameShort string  `json:"server_name_short"`
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	ReservedSlots   int     `json:"reserved_slots"`
	Password        string  `json:"password"`
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
		if !httphelper.Bind(ctx, &req) {
			return
		}

		server.ShortName = req.ServerNameShort
		server.Name = req.ServerName
		server.Address = req.Host
		server.Port = req.Port
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Password = req.Password
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled
		server.LogSecret = req.LogSecret
		server.EnableStats = req.EnableStats

		if errSave := h.serversUsecase.SaveServer(ctx, &server); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to update server", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		slog.Info("ServerStore config updated",
			slog.Int("server_id", server.ServerID),
			slog.String("name", server.ShortName))
	}
}

func (h *serversHandler) onAPIGetServersAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter := domain.ServerQueryFilter{
			IncludeDisabled: true,
		}

		servers, _, errServers := h.serversUsecase.GetServers(ctx, filter)
		if errServers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, servers)
	}
}

func (h *serversHandler) onAPIPostServerDelete() gin.HandlerFunc {
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
			slog.Error("Failed to delete server", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		slog.Info("ServerStore config deleted",
			slog.Int("server_id", server.ServerID),
			slog.String("name", server.ShortName))
	}
}
