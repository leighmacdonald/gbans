package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	"go.uber.org/zap"
)

type (
	UpdateA2SHandler    func(serverID int, newState *a2s.ServerInfo)
	UpdateStatusHandler func(serverID int, newState extra.Status)
	UpdateMSLHandler    func(newState []ServerLocation)
)

type ServerStateCollector struct {
	log              *zap.Logger
	onUpdateA2S      UpdateA2SHandler
	onUpdateStatus   UpdateStatusHandler
	onUpdateMSL      UpdateMSLHandler
	statusUpdateFreq time.Duration
	msListUpdateFreq time.Duration
}

func NewServerStateCollector(logger *zap.Logger, onUpdate UpdateA2SHandler, onUpdateStatus UpdateStatusHandler, onUpdateMSL UpdateMSLHandler) *ServerStateCollector {
	const statusUpdateFreq = time.Second * 30

	const msListUpdateFreq = time.Minute

	return &ServerStateCollector{
		log:              logger,
		onUpdateA2S:      onUpdate,
		onUpdateStatus:   onUpdateStatus,
		onUpdateMSL:      onUpdateMSL,
		statusUpdateFreq: statusUpdateFreq,
		msListUpdateFreq: msListUpdateFreq,
	}
}

func (c *ServerStateCollector) startMSL(ctx context.Context) {
	var (
		log              = c.log.Named("msl_update")
		mlUpdateTicker   = time.NewTicker(c.msListUpdateFreq)
		masterServerList []ServerLocation
	)

	for {
		select {
		case <-mlUpdateTicker.C:
			newMsl, errUpdateMsl := c.updateMSL(ctx)
			if errUpdateMsl != nil {
				log.Error("Failed to update master server list", zap.Error(errUpdateMsl))

				continue
			}

			masterServerList = newMsl

			go c.onUpdateMSL(masterServerList)
		case <-ctx.Done():
			return
		}
	}
}

func (c *ServerStateCollector) startStatus(ctx context.Context, configs []ServerConfig) {
	const timeout = time.Second * 15

	var (
		log                = c.log.Named("status_update")
		statusUpdateTicker = time.NewTicker(c.statusUpdateFreq)
	)

	for {
		select {
		case <-statusUpdateTicker.C:
			for _, serverConfig := range configs {
				go func(conf ServerConfig) {
					console, errDial := rcon.Dial(ctx, conf.addr(), conf.Password, timeout)

					if errDial != nil {
						log.Error("Failed to dial rcon", zap.String("err", errDial.Error()))

						return
					}

					resp, errRcon := console.Exec("status")
					if errRcon != nil {
						log.Error("Failed to exec rcon status", zap.String("server", conf.Name), zap.Error(errRcon))

						return
					}

					status, errParse := extra.ParseStatus(resp, true)
					if errParse != nil {
						log.Error("Failed to parse rcon status", zap.Error(errParse))

						return
					}

					go c.onUpdateStatus(conf.ServerID, status)

					log.Debug("Updated", zap.String("server", conf.Name))
				}(serverConfig)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *ServerStateCollector) startA2S(ctx context.Context, configs []ServerConfig) {
	const timeout = time.Second * 10

	var (
		log      = c.log.Named("a2s_update")
		a2sTimer = time.NewTicker(c.statusUpdateFreq)
	)

	for {
		select {
		case <-a2sTimer.C:
			for _, serverConfig := range configs {
				go func(conf ServerConfig) {
					client, errClient := a2s.NewClient(conf.addr(), a2s.SetMaxPacketSize(14000), a2s.TimeoutOption(timeout))
					if errClient != nil {
						log.Error("Failed to create a2s client", zap.String("server", conf.Name), zap.Error(errClient))

						return
					}

					defer func() {
						if errClose := client.Close(); errClose != nil {
							log.Error("Failed to close a2s conn", zap.String("server", conf.Name), zap.Error(errClose))
						}
					}()

					result, errQuery := client.QueryInfo()
					if errQuery != nil {
						log.Debug("Failed to query a2s server", zap.String("server", conf.Name), zap.String("err", errQuery.Error()))

						return
					}

					go c.onUpdateA2S(conf.ServerID, result)

					log.Debug("Updated", zap.String("server", conf.Name))
				}(serverConfig)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *ServerStateCollector) Start(ctx context.Context, configs []ServerConfig) {
	go c.startMSL(ctx)
	go c.startA2S(ctx, configs)
	go c.startStatus(ctx, configs)
}

func NewServerConfig(serverID int, name string, address string, port int, password string) ServerConfig {
	return ServerConfig{
		ServerID: serverID,
		Name:     name,
		Host:     address,
		Port:     port,
		Password: password,
	}
}

type ServerConfig struct {
	ServerID int
	Name     string
	Host     string
	Port     int
	Password string
}

func (config *ServerConfig) addr() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

type SvRegion int

const (
	RegionNaEast SvRegion = iota
	RegionNAWest
	RegionSouthAmerica
	RegionEurope
	RegionAsia
	RegionAustralia
	RegionMiddleEast
	RegionAfrica
	RegionWorld SvRegion = 255
)

func SteamRegionIDString(region SvRegion) string {
	switch region {
	case RegionNaEast:
		return "ne"
	case RegionNAWest:
		return "nw"
	case RegionSouthAmerica:
		return "sa"
	case RegionEurope:
		return "eu"
	case RegionAsia:
		return "as"
	case RegionAustralia:
		return "au"
	case RegionMiddleEast:
		return "me"
	case RegionAfrica:
		return "af"
	case RegionWorld:
		fallthrough
	default:
		return "wd"
	}
}

func (c *ServerStateCollector) updateMSL(ctx context.Context) ([]ServerLocation, error) {
	allServers, errServers := steamweb.GetServerList(ctx, map[string]string{
		"appid":     "440",
		"dedicated": "1",
	})

	if errServers != nil {
		return nil, errors.Wrap(errServers, "Failed to fetch updated list")
	}

	var ( //nolint:prealloc
		communityServers []ServerLocation
		stats            = NewGlobalTF2Stats()
	)

	for _, baseServer := range allServers {
		server := ServerLocation{
			LatLong: ip2location.LatLong{},
			Server:  baseServer,
		}

		stats.ServersTotal++
		stats.Players += server.Players
		stats.Bots += server.Bots

		switch {
		case server.MaxPlayers > 0 && server.Players >= server.MaxPlayers:
			stats.CapacityFull++
		case server.Players == 0:
			stats.CapacityEmpty++
		default:
			stats.CapacityPartial++
		}

		if server.Secure {
			stats.Secure++
		}

		region := SteamRegionIDString(SvRegion(server.Region))

		_, regionFound := stats.Regions[region]
		if !regionFound {
			stats.Regions[region] = 0
		}

		stats.Regions[region] += server.Players

		mapType := GuessMapType(server.Map)

		_, mapTypeFound := stats.MapTypes[mapType]
		if !mapTypeFound {
			stats.MapTypes[mapType] = 0
		}

		stats.MapTypes[mapType]++
		if strings.Contains(server.GameType, "valve") ||
			!server.Dedicated ||
			!server.Secure {
			stats.ServersCommunity++

			continue
		}

		communityServers = append(communityServers, server)
	}

	return communityServers, nil
}

func GuessMapType(mapName string) string {
	mapName = strings.TrimPrefix(mapName, "workshop/")
	pieces := strings.SplitN(mapName, "_", 2)

	if len(pieces) == 1 {
		return "unknown"
	}

	return strings.ToLower(pieces[0])
}

type GlobalTF2StatsSnapshot struct {
	StatID           int64          `json:"stat_id"`
	Players          int            `json:"players"`
	Bots             int            `json:"bots"`
	Secure           int            `json:"secure"`
	ServersCommunity int            `json:"servers_community"`
	ServersTotal     int            `json:"servers_total"`
	CapacityFull     int            `json:"capacity_full"`
	CapacityEmpty    int            `json:"capacity_empty"`
	CapacityPartial  int            `json:"capacity_partial"`
	MapTypes         map[string]int `json:"map_types"`
	Regions          map[string]int `json:"regions"`
	CreatedOn        time.Time      `json:"created_on"`
}

func (stats GlobalTF2StatsSnapshot) TrimMapTypes() map[string]int {
	const minSize = 5

	out := map[string]int{}

	for keyKey, value := range stats.MapTypes {
		mapKey := keyKey
		if value < minSize {
			mapKey = "unknown"
		}

		out[mapKey] = value
	}

	return out
}

func NewGlobalTF2Stats() GlobalTF2StatsSnapshot {
	return GlobalTF2StatsSnapshot{
		MapTypes:  map[string]int{},
		Regions:   map[string]int{},
		CreatedOn: time.Now(),
	}
}

type ServerLocation struct {
	ip2location.LatLong
	steamweb.Server
}
