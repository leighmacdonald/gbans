package servers

import (
	"errors"
	"math"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/maruel/natural"
)

type serversHandler struct {
	servers domain.ServersUsecase
	state   domain.StateUsecase
}

func NewHandler(engine *gin.Engine, serversUsecase domain.ServersUsecase, stateUsecase domain.StateUsecase, ath domain.AuthUsecase) {
	handler := &serversHandler{
		servers: serversUsecase,
		state:   stateUsecase,
	}

	engine.GET("/api/servers/state", handler.onAPIGetServerStates())
	engine.GET("/api/servers", handler.onAPIGetServers())

	// admin
	srvGrp := engine.Group("/")
	{
		admin := srvGrp.Use(ath.Middleware(domain.PAdmin))
		admin.POST("/api/servers", handler.onAPIPostServer())
		admin.POST("/api/servers/:server_id", handler.onAPIPostServerUpdate())
		admin.DELETE("/api/servers/:server_id", handler.onAPIPostServerDelete())
		admin.GET("/api/servers_admin", handler.onAPIGetServersAdmin())
	}
}

func (h *serversHandler) onAPIGetServers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := h.servers.Servers(ctx, domain.ServerQueryFilter{})
		if errServers != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errServers, domain.ErrInternal)))

			return
		}

		var servers []domain.ServerInfoSafe
		for _, server := range fullServers {
			servers = append(servers, domain.ServerInfoSafe{
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
			curState = h.state.Current()
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
				Humans:     srv.Humans,
				Map:        srv.Map,
				GameTypes:  []string{},
				Latitude:   srv.Latitude,
				Longitude:  srv.Longitude,
				Distance:   distance(srv.Latitude, srv.Longitude, lat, lon),
			})
		}

		sort.Slice(servers, func(i, j int) bool {
			return natural.Less(servers[i].Name, servers[j].Name)
		})

		if servers == nil {
			servers = []domain.SafeServer{}
		}

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
		var req domain.RequestServerUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		server, errSave := h.servers.Save(ctx, req)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, server)
	}
}

func (h *serversHandler) onAPIPostServerUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		serverID, idFound := httphelper.GetIntParam(ctx, "server_id")
		if !idFound {
			return
		}

		var req domain.RequestServerUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.ServerID = serverID

		server, errSave := h.servers.Save(ctx, req)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, server)
	}
}

func (h *serversHandler) onAPIGetServersAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter := domain.ServerQueryFilter{
			IncludeDisabled: true,
		}

		servers, _, errServers := h.servers.Servers(ctx, filter)
		if errServers != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errServers, domain.ErrInternal)))

			return
		}

		if servers == nil {
			servers = []domain.Server{}
		}

		ctx.JSON(http.StatusOK, servers)
	}
}

func (h *serversHandler) onAPIPostServerDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		serverID, idFound := httphelper.GetIntParam(ctx, "server_id")
		if !idFound {
			return
		}

		if serverID == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		if err := h.servers.Delete(ctx, serverID); err != nil {
			switch {
			case errors.Is(err, domain.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, errors.Join(err, domain.ErrNotFound)))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
