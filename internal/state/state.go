package state

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
	connections      map[string]*rconController
	connectionsMu    *sync.RWMutex
}

func NewServerStateCollector(logger *zap.Logger, onUpdate UpdateA2SHandler, onUpdateStatus UpdateStatusHandler, onUpdateMSL UpdateMSLHandler) *ServerStateCollector {
	const (
		statusUpdateFreq = time.Minute
		msListUpdateFreq = time.Minute
	)

	return &ServerStateCollector{
		log:              logger,
		onUpdateA2S:      onUpdate,
		onUpdateStatus:   onUpdateStatus,
		onUpdateMSL:      onUpdateMSL,
		statusUpdateFreq: statusUpdateFreq,
		msListUpdateFreq: msListUpdateFreq,
		connections:      map[string]*rconController{},
		connectionsMu:    &sync.RWMutex{},
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

type rconController struct {
	*rcon.RemoteConsole
	attempts           int
	lastConnectSuccess time.Time
	lastConnectAttempt time.Time
}

func (rc rconController) connected() bool {
	return rc.RemoteConsole != nil
}

func (rc rconController) allowedToConnect() bool {
	const (
		waitInterval    = time.Minute
		limitMultiCount = 10
	)

	if rc.attempts == 0 {
		return true
	}

	multi := rc.attempts
	if multi > limitMultiCount {
		multi = limitMultiCount
	}

	return rc.lastConnectAttempt.Add(waitInterval * time.Duration(multi)).Before(time.Now())
}

func (c *ServerStateCollector) startStatus(ctx context.Context, configs []ServerConfig) {
	const timeout = time.Second * 15

	var (
		logger             = c.log.Named("status_update")
		statusUpdateTicker = time.NewTicker(c.statusUpdateFreq)
	)

	for {
		select {
		case <-statusUpdateTicker.C:
			waitGroup := &sync.WaitGroup{}
			startTIme := time.Now()
			successful := atomic.Int32{}

			for _, serverConfig := range configs {
				waitGroup.Add(1)

				go func(conf ServerConfig) {
					defer waitGroup.Done()

					addr := conf.addr()
					connected := false
					allowed := false

					log := logger.Named(conf.Name)

					c.connectionsMu.Lock()
					controller, found := c.connections[addr]

					if !found {
						controller = &rconController{RemoteConsole: nil, attempts: 0, lastConnectAttempt: time.Now()}
						c.connections[addr] = controller
					} else {
						connected = controller.connected()
					}

					allowed = controller.allowedToConnect()

					c.connectionsMu.Unlock()

					if !allowed {
						log.Debug("Delaying connect")
					}

					if !connected {
						newConsole, errDial := rcon.Dial(ctx, addr, conf.Password, timeout)

						if errDial != nil {
							log.Debug("Failed to dial rcon", zap.String("err", errDial.Error()))
						}

						c.connectionsMu.Lock()
						controller.lastConnectAttempt = time.Now()

						if newConsole != nil {
							controller.lastConnectSuccess = controller.lastConnectAttempt
							controller.attempts = 0
							controller.RemoteConsole = newConsole
							connected = true
						} else {
							controller.attempts++
						}
						c.connectionsMu.Unlock()
					}

					if !connected {
						return
					}

					resp, errRcon := controller.Exec("status")
					if errRcon != nil {
						log.Error("Failed to exec rcon status", zap.String("server", conf.Name), zap.Error(errRcon))

						c.connectionsMu.Lock()
						if errClose := controller.RemoteConsole.Close(); errClose != nil {
							log.Error("Failed to close rcon connection", zap.String("server", conf.Name), zap.Error(errClose))
						}

						controller.RemoteConsole = nil
						c.connectionsMu.Unlock()

						return
					}

					status, errParse := extra.ParseStatus(resp, true)
					if errParse != nil {
						log.Error("Failed to parse rcon status", zap.Error(errParse))

						return
					}

					go c.onUpdateStatus(conf.ServerID, status)

					log.Debug("Updated", zap.String("server", conf.Name))
					successful.Add(1)
				}(serverConfig)

				waitGroup.Wait()
				logger.Info("RCON update cycle complete",
					zap.Int32("success", successful.Load()),
					zap.Int32("fail", int32(len(configs))-successful.Load()),
					zap.Duration("duration", time.Since(startTIme)))
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
