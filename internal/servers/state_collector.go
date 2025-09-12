package servers

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v4/extra"
	"golang.org/x/exp/slices"
)

var (
	ErrRCONCommand      = errors.New("failed to execute rcon command")
	ErrFailedToDialRCON = errors.New("failed to dial rcon")
)

type Collector struct {
	statusUpdateFreq time.Duration
	updateTimeout    time.Duration
	serverState      map[int]ServerState
	stateMu          *sync.RWMutex
	configs          []ServerConfig
	configMu         *sync.RWMutex
	maxPlayersRx     *regexp.Regexp
	playersRx        *regexp.Regexp
	serverUsecase    ServersUsecase
}

func NewCollector(serverUsecase ServersUsecase) *Collector {
	const (
		statusUpdateFreq = time.Second * 20
		updateTimeout    = time.Second * 5
	)

	return &Collector{
		statusUpdateFreq: statusUpdateFreq,
		updateTimeout:    updateTimeout,
		serverState:      map[int]ServerState{},
		stateMu:          &sync.RWMutex{},
		configMu:         &sync.RWMutex{},
		maxPlayersRx:     regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`),
		playersRx:        regexp.MustCompile(`players\s: (\d+)\s+humans,\s+(\d+)\s+bots\s\((\d+)\s+max`),
		serverUsecase:    serverUsecase,
	}
}

func (c *Collector) Configs() []ServerConfig {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	var conf []ServerConfig

	conf = append(conf, c.configs...)

	return conf
}

func (c *Collector) ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error) {
	conn, errConn := rcon.Dial(ctx, addr, password, time.Second*5)
	if errConn != nil {
		return "", errors.Join(errConn, ErrFailedToDialRCON)
	}

	resp, errExec := conn.Exec(cmd)
	if errExec != nil {
		return "", errors.Join(errExec, ErrRCONExecCommand)
	}

	if errClose := conn.Close(); errClose != nil {
		slog.Error("Could not close rcon connection", log.ErrAttr(errClose))
	}

	return resp, nil
}

func (c *Collector) GetServer(serverID int) (ServerConfig, error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	configs := c.Configs()

	serverIdx := slices.IndexFunc(configs, func(serverConfig ServerConfig) bool {
		return serverConfig.ServerID == serverID
	})

	if serverIdx == -1 {
		return ServerConfig{}, ErrUnknownServerID
	}

	return configs[serverIdx], nil
}

func (c *Collector) Current() []ServerState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	var curState []ServerState //nolint:prealloc
	for _, s := range c.serverState {
		curState = append(curState, s)
	}

	sort.SliceStable(curState, func(i, j int) bool {
		return curState[i].Name < curState[j].Name
	})

	return curState
}

func (c *Collector) onStatusUpdate(conf ServerConfig, newState Status, maxVisible int) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	server := c.serverState[conf.ServerID]
	server.PlayerCount = newState.PlayersCount
	// if newState.IPInfo.SDR {
	// 	server.IP = newState.IPInfo.FakeIP
	// 	server.Port = uint16(newState.IPInfo.FakePort) //nolint:gosec
	// 	server.IPPublic = newState.IPInfo.PublicIP
	// 	server.PortPublic = uint16(newState.IPInfo.PublicPort) //nolint:gosec
	// } else {
	// 	server.IP = newState.IPInfo.PublicIP
	// 	server.Port = uint16(newState.IPInfo.PublicPort) //nolint:gosec
	// 	server.IPPublic = newState.IPInfo.PublicIP
	// 	server.PortPublic = uint16(newState.IPInfo.PublicPort) //nolint:gosec
	// }

	server.STVIP = newState.IPInfo.SourceTVIP
	server.STVPort = uint16(newState.IPInfo.SourceTVFPort) //nolint:gosec

	if maxVisible >= 0 {
		server.MaxPlayers = maxVisible
	} else {
		server.MaxPlayers = newState.PlayersMax
	}

	if newState.ServerName != "" {
		server.Name = newState.ServerName
	}

	server.Version = newState.Version
	server.Edicts = newState.Edicts
	server.Tags = newState.Tags

	if newState.Map != "" && newState.Map != server.Map {
		server.Map = newState.Map
	}

	server.Players = newState.Players
	server.Humans = newState.Humans
	server.Bots = newState.Bots

	c.serverState[conf.ServerID] = server
}

func (c *Collector) setServerConfigs(ctx context.Context, configs []ServerConfig) {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	var gone []ServerConfig

	for _, exist := range c.configs {
		exists := false

		for _, newConf := range configs {
			if exist.ServerID == newConf.ServerID {
				exists = true

				break
			}
		}

		if !exists {
			gone = append(gone, exist)
		}
	}

	for _, conf := range gone {
		delete(c.serverState, conf.ServerID)
	}

	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	for _, cfg := range configs {
		if _, found := c.serverState[cfg.ServerID]; !found {
			if !cfg.Enabled {
				continue
			}

			addr, errResolve := ResolveIP(ctx, cfg.Host)
			if errResolve != nil {
				slog.Warn("Failed to resolve server ip", slog.String("addr", addr), log.ErrAttr(errResolve))
				addr = cfg.Host
			}

			c.serverState[cfg.ServerID] = ServerState{
				ServerID:      cfg.ServerID,
				Name:          cfg.DefaultHostname,
				NameShort:     cfg.Tag,
				Host:          cfg.Host,
				Port:          cfg.Port,
				Enabled:       cfg.Enabled,
				RconPassword:  cfg.RconPassword,
				ReservedSlots: cfg.ReservedSlots,
				CC:            cfg.CC,
				Region:        cfg.Region,
				Latitude:      cfg.Latitude,
				Longitude:     cfg.Longitude,
				IP:            addr,
			}
		}
	}

	c.configs = configs
}

func (c *Collector) Update(serverID int, update PartialStateUpdate) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	curState, ok := c.serverState[serverID]
	if !ok {
		return ErrUnknownServer
	}

	if update.Hostname != "" {
		curState.Name = update.Hostname
	}

	curState.Map = update.CurrentMap
	curState.PlayerCount = update.PlayersReal
	curState.MaxPlayers = update.PlayersVisible
	curState.Bots = update.PlayersTotal - update.PlayersReal
	c.serverState[serverID] = curState

	return nil
}

type Status struct {
	extra.Status
	Humans int
	Bots   int
}

var (
	ErrStatusParse       = errors.New("failed to parse status response")
	ErrMaxPlayerIntParse = errors.New("failed to cast max players value")
	ErrMaxPlayerParse    = errors.New("failed to parse sv_visiblemaxplayers response")
	ErrDNSResolve        = errors.New("failed to resolve server dns")
	ErrRCONExecCommand   = errors.New("failed to perform command")
)

func (c *Collector) status(ctx context.Context, serverID int) (Status, error) {
	server, errServerID := c.GetServer(serverID)
	if errServerID != nil {
		return Status{}, errServerID
	}

	statusResp, errStatus := c.ExecRaw(ctx, server.Addr(), server.RconPassword, "status")
	if errStatus != nil {
		return Status{}, errStatus
	}

	pStatus, errParse := extra.ParseStatus(statusResp, true)
	if errParse != nil {
		return Status{}, errors.Join(errParse, ErrStatusParse)
	}

	status := Status{Status: pStatus}
	matches := c.playersRx.FindStringSubmatch(statusResp)
	if len(matches) > 0 {
		players, errPlayers := strconv.Atoi(matches[1])
		if errPlayers != nil {
			return Status{}, ErrStatusParse
		}
		status.Humans = players

		bots, errBots := strconv.Atoi(matches[2])
		if errBots != nil {
			return Status{}, ErrStatusParse
		}
		status.Bots = bots

		maxPlayers, errMaxPlayers := strconv.Atoi(matches[3])
		if errMaxPlayers != nil {
			return Status{}, ErrStatusParse
		}

		if maxPlayers%2 != 0 {
			// Assume that if we have an uneven player count that it's a SourceTV instance and ignore it.
			status.Bots--
			maxPlayers--
		}

		status.PlayersMax = maxPlayers
	}

	return status, nil
}

const maxPlayersSupported = 101

func (c *Collector) maxVisiblePlayers(ctx context.Context, serverID int) (int, error) {
	server, errServerID := c.GetServer(serverID)
	if errServerID != nil {
		return 0, errServerID
	}

	maxPlayersResp, errMaxPlayers := c.ExecRaw(ctx, server.Addr(), server.RconPassword, "sv_visiblemaxplayers")
	if errMaxPlayers != nil {
		return 0, errMaxPlayers
	}

	matches := c.maxPlayersRx.FindStringSubmatch(maxPlayersResp)
	if matches == nil || len(matches) != 2 {
		return 0, ErrMaxPlayerParse
	}

	maxPlayers, errCast := strconv.ParseInt(matches[1], 10, 32)
	if errCast != nil {
		return 0, errors.Join(errCast, ErrMaxPlayerIntParse)
	}

	if maxPlayers > maxPlayersSupported {
		maxPlayers = -1
	}

	return int(maxPlayers), nil
}

func (c *Collector) startStatus(ctx context.Context) {
	statusUpdateTicker := time.NewTicker(c.statusUpdateFreq)

	for {
		select {
		case <-statusUpdateTicker.C:
			waitGroup := &sync.WaitGroup{}
			successful := atomic.Int32{}
			existing := atomic.Int32{}

			c.stateMu.RLock()
			configs := c.configs
			c.stateMu.RUnlock()

			startTIme := time.Now()

			for _, serverConfigInstance := range configs {
				waitGroup.Add(1)

				go func(lCtx context.Context, conf ServerConfig) {
					defer waitGroup.Done()

					status, errStatus := c.status(lCtx, conf.ServerID)
					if errStatus != nil {
						slog.Error("Failed to parse status", slog.String("error", errStatus.Error()))

						return
					}

					maxVisible, errMaxVisible := c.maxVisiblePlayers(lCtx, conf.ServerID)
					if errMaxVisible != nil {
						slog.Warn("Got invalid max players value", log.ErrAttr(errMaxVisible), slog.Int("server_id", conf.ServerID))
					}

					c.onStatusUpdate(conf, status, maxVisible)

					successful.Add(1)
				}(ctx, serverConfigInstance)
			}

			waitGroup.Wait()

			fail := len(configs) - int(successful.Load())

			if fail > 0 {
				slog.Debug("RCON update cycle complete",
					slog.Int("success", int(successful.Load())),
					slog.Int("existing", int(existing.Load())),
					slog.Int("fail", fail),
					slog.Duration("duration", time.Since(startTIme)))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Collector) updateServerConfigs(ctx context.Context) {
	servers, _, errServers := c.serverUsecase.Servers(ctx, ServerQueryFilter{
		QueryFilter:     domain.QueryFilter{Deleted: false},
		IncludeDisabled: false,
	})

	if errServers != nil && !errors.Is(errServers, database.ErrNoResult) {
		slog.Error("Failed to fetch servers, cannot update State", log.ErrAttr(errServers))

		return
	}

	configs := make([]ServerConfig, len(servers))

	for i, server := range servers {
		configs[i] = newServerConfig(
			server.ServerID,
			server.ShortName,
			server.Name,
			server.Address,
			server.Port,
			server.RCON,
			server.ReservedSlots,
			server.CC,
			server.Region,
			server.Latitude,
			server.Longitude,
			server.IsEnabled,
		)
	}

	c.setServerConfigs(ctx, configs)
}

func (c *Collector) Start(ctx context.Context) {
	updateTicker := time.NewTicker(time.Minute * 30)

	c.updateServerConfigs(ctx)

	go c.startStatus(ctx)

	for {
		select {
		case <-updateTicker.C:
			c.updateServerConfigs(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func newServerConfig(serverID int, name string, defaultHostname string, address string,
	port uint16, rconPassword string, reserved int, countryCode string, region string, lat float64, long float64,
	enabled bool,
) ServerConfig {
	return ServerConfig{
		ServerID:        serverID,
		Tag:             name,
		DefaultHostname: defaultHostname,
		Host:            address,
		Port:            port,
		Enabled:         enabled,
		RconPassword:    rconPassword,
		ReservedSlots:   reserved,
		CC:              countryCode,
		Region:          region,
		Latitude:        lat,
		Longitude:       long,
	}
}

func ResolveIP(ctx context.Context, addr string) (string, error) {
	ipAddr := net.ParseIP(addr)
	if ipAddr != nil {
		return ipAddr.String(), nil
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
	if err != nil || len(ips) == 0 {
		return "", errors.Join(err, ErrDNSResolve)
	}

	return ips[0].String(), nil
}
