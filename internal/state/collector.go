package state

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

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v4/extra"
	"golang.org/x/exp/slices"
)

var (
	ErrRCONCommand      = errors.New("failed to execute rcon command")
	ErrFailedToDialRCON = errors.New("failed to connect to conf")
)

type Collector struct {
	statusUpdateFreq time.Duration
	updateTimeout    time.Duration
	serverState      map[int]domain.ServerState
	stateMu          *sync.RWMutex
	configs          []domain.ServerConfig
	configMu         *sync.RWMutex
	maxPlayersRx     *regexp.Regexp
	serverUsecase    domain.ServersUsecase
}

func NewCollector(serverUsecase domain.ServersUsecase) *Collector {
	const (
		statusUpdateFreq = time.Second * 20
		updateTimeout    = time.Second * 5
	)

	return &Collector{
		statusUpdateFreq: statusUpdateFreq,
		updateTimeout:    updateTimeout,
		serverState:      map[int]domain.ServerState{},
		stateMu:          &sync.RWMutex{},
		configMu:         &sync.RWMutex{},
		maxPlayersRx:     regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`),
		serverUsecase:    serverUsecase,
	}
}

func (c *Collector) Configs() []domain.ServerConfig {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	var conf []domain.ServerConfig

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

func (c *Collector) GetServer(serverID int) (domain.ServerConfig, error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	configs := c.Configs()

	serverIdx := slices.IndexFunc(configs, func(serverConfig domain.ServerConfig) bool {
		return serverConfig.ServerID == serverID
	})

	if serverIdx == -1 {
		return domain.ServerConfig{}, domain.ErrUnknownServerID
	}

	return configs[serverIdx], nil
}

func (c *Collector) Current() []domain.ServerState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	var curState []domain.ServerState //nolint:prealloc
	for _, s := range c.serverState {
		curState = append(curState, s)
	}

	sort.SliceStable(curState, func(i, j int) bool {
		return curState[i].Name < curState[j].Name
	})

	return curState
}

func (c *Collector) onStatusUpdate(conf domain.ServerConfig, newState extra.Status, maxVisible int) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	server := c.serverState[conf.ServerID]
	server.PlayerCount = newState.PlayersCount

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

	c.serverState[conf.ServerID] = server
}

func (c *Collector) setServerConfigs(configs []domain.ServerConfig) {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	var gone []domain.ServerConfig

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
			addr, errResolve := ResolveIP(cfg.Host)
			if errResolve != nil {
				slog.Warn("Failed to resolve server ip", slog.String("addr", addr), log.ErrAttr(errResolve))
				addr = cfg.Host
			}

			c.serverState[cfg.ServerID] = domain.ServerState{
				ServerID:      cfg.ServerID,
				Name:          cfg.DefaultHostname,
				NameShort:     cfg.Tag,
				Host:          cfg.Host,
				Port:          cfg.Port,
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

func (c *Collector) Update(serverID int, update domain.PartialStateUpdate) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	curState, ok := c.serverState[serverID]
	if !ok {
		return domain.ErrUnknownServer
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

var (
	ErrStatusParse       = errors.New("failed to parse status response")
	ErrMaxPlayerIntParse = errors.New("failed to cast max players value")
	ErrMaxPlayerParse    = errors.New("failed to parse sv_visiblemaxplayers response")
	ErrDNSResolve        = errors.New("failed to resolve server dns")
	ErrRCONExecCommand   = errors.New("failed to perform command")
)

func (c *Collector) status(ctx context.Context, serverID int) (extra.Status, error) {
	server, errServerID := c.GetServer(serverID)
	if errServerID != nil {
		return extra.Status{}, errServerID
	}

	statusResp, errStatus := c.ExecRaw(ctx, server.Addr(), server.RconPassword, "status")
	if errStatus != nil {
		return extra.Status{}, errStatus
	}

	status, errParse := extra.ParseStatus(statusResp, true)
	if errParse != nil {
		return extra.Status{}, errors.Join(errParse, ErrStatusParse)
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

				go func(lCtx context.Context, conf domain.ServerConfig) {
					defer waitGroup.Done()

					status, errStatus := c.status(lCtx, conf.ServerID)
					if errStatus != nil {
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

func (c *Collector) Start(ctx context.Context) {
	var (
		trigger      = make(chan any)
		updateTicker = time.NewTicker(time.Minute * 30)
	)

	go c.startStatus(ctx)

	go func() {
		trigger <- true
	}()

	for {
		select {
		case <-updateTicker.C:
			trigger <- true
		case <-trigger:
			servers, _, errServers := c.serverUsecase.GetServers(ctx, domain.ServerQueryFilter{
				QueryFilter:     domain.QueryFilter{Deleted: false},
				IncludeDisabled: false,
			})
			if errServers != nil && !errors.Is(errServers, domain.ErrNoResult) {
				slog.Error("Failed to fetch servers, cannot update State", log.ErrAttr(errServers))

				continue
			}

			var configs []domain.ServerConfig
			for _, server := range servers {
				configs = append(configs, newServerConfig(
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
				))
			}

			c.setServerConfigs(configs)
		case <-ctx.Done():
			return
		}
	}
}

func newServerConfig(serverID int, name string, defaultHostname string, address string,
	port int, rconPassword string, reserved int, countryCode string, region string, lat float64, long float64,
) domain.ServerConfig {
	return domain.ServerConfig{
		ServerID:        serverID,
		Tag:             name,
		DefaultHostname: defaultHostname,
		Host:            address,
		Port:            port,
		RconPassword:    rconPassword,
		ReservedSlots:   reserved,
		CC:              countryCode,
		Region:          region,
		Latitude:        lat,
		Longitude:       long,
	}
}

func ResolveIP(addr string) (string, error) {
	ipAddr := net.ParseIP(addr)
	if ipAddr != nil {
		return ipAddr.String(), nil
	}

	ips, err := net.LookupIP(addr)
	if err != nil || len(ips) == 0 {
		return "", errors.Join(err, ErrDNSResolve)
	}

	return ips[0].String(), nil
}
