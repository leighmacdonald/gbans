package app

import (
	"context"
	gerrors "errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
)

// serverDetails contains the entire state for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type serverDetails struct {
	// Database
	ServerID  int    `json:"server_id"`
	NameShort string `json:"name_short"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	// IP is a distinct entry vs host since steam no longer allows steam:// protocol handler links to use a fqdn
	IP            string    `json:"ip"`
	Port          int       `json:"port"`
	Enabled       bool      `json:"enabled"`
	Region        string    `json:"region"`
	CC            string    `json:"cc"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Reserved      int       `json:"reserved"`
	LastUpdate    time.Time `json:"last_update"`
	ReservedSlots int       `json:"reserved_slots"`
	Protocol      uint8     `json:"protocol"`
	RconPassword  string    `json:"rcon_password"`
	EnableStats   bool      `json:"enable_stats"`
	Map           string    `json:"map"`
	// Name of the folder containing the game files.
	Folder string `json:"folder"`
	// Full name of the game.
	Game string `json:"game"`
	// Steam Application ID of game.
	AppID uint16 `json:"app_id"`
	// Number of players on the server.
	PlayerCount int `json:"player_count"`
	// Maximum number of players the server reports it can hold.
	MaxPlayers int `json:"max_players"`
	// Number of bots on the server.
	Bots int `json:"bots"`
	// Indicates whether the server requires a password
	Password bool `json:"password"`
	// Specifies whether the server uses VAC
	VAC bool `json:"vac"`
	// Version of the game installed on the server.
	Version string `json:"version"`
	// Server's SteamID.
	SteamID steamid.SID64 `json:"steam_id"`
	// Tags that describe the game according to the server (for future use.)
	Keywords []string `json:"keywords"`
	Edicts   []int    `json:"edicts"`
	// The server's 64-bit GameID. If this is present, a more accurate AppID is present in the low 24 bits.
	// The earlier AppID could have been truncated as it was forced into 16-bit storage.
	GameID uint64 `json:"game_id"` // Needed?
	// Spectator port number for SourceTV.
	STVPort uint16 `json:"stv_port"`
	// Name of the spectator server for SourceTV.
	STVName string `json:"stv_name"`

	Tags    []string       `json:"tags"`
	Players []extra.Player `json:"players"`
}

type baseServer struct {
	ServerID   int      `json:"server_id"`
	Host       string   `json:"host"`
	Port       int      `json:"port"`
	IP         string   `json:"ip"`
	Name       string   `json:"name"`
	NameShort  string   `json:"name_short"`
	Region     string   `json:"region"`
	CC         string   `json:"cc"`
	Players    int      `json:"players"`
	MaxPlayers int      `json:"max_players"`
	Bots       int      `json:"bots"`
	Map        string   `json:"map"`
	GameTypes  []string `json:"game_types"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Distance   float64  `json:"distance"`
}

var errUnknownServer = errors.New("Unknown server")

type serverDetailsCollection []serverDetails

type playerServerInfo struct {
	Player   extra.Player
	ServerID int
}

type findOpts struct {
	Name    string
	IP      *net.IP
	SteamID steamid.SID64
	CIDR    *net.IPNet `json:"cidr"`
}

func (c *serverDetailsCollection) find(opts findOpts) []playerServerInfo {
	var found []playerServerInfo

	for _, server := range *c {
		for _, player := range server.Players {
			matched := false
			if opts.SteamID.Valid() && player.SID == opts.SteamID {
				matched = true
			}

			if opts.Name != "" {
				queryName := opts.Name
				if !strings.HasPrefix(queryName, "*") {
					queryName = "*" + queryName
				}

				if !strings.HasSuffix(queryName, "*") {
					queryName += "*"
				}

				m := glob.Glob(strings.ToLower(queryName), strings.ToLower(player.Name))
				if m {
					matched = true
				}
			}

			if opts.IP != nil && opts.IP.Equal(player.IP) {
				matched = true
			}

			if opts.CIDR != nil && opts.CIDR.Contains(player.IP) {
				matched = true
			}

			if matched {
				found = append(found, playerServerInfo{Player: player, ServerID: server.ServerID})
			}
		}
	}

	return found
}

func (c *serverDetailsCollection) sortRegion() map[string]serverDetailsCollection {
	serverMap := map[string]serverDetailsCollection{}
	for _, server := range *c {
		_, exists := serverMap[server.Region]
		if !exists {
			serverMap[server.Region] = serverDetailsCollection{}
		}

		serverMap[server.Region] = append(serverMap[server.Region], server)
	}

	return serverMap
}

func (c *serverDetailsCollection) byServerID(serverID int) (serverDetails, bool) {
	for _, server := range *c {
		if server.ServerID == serverID {
			return server, true
		}
	}

	return serverDetails{}, false
}

func (c *serverDetailsCollection) byName(name string, wildcardOk bool) serverDetailsCollection {
	var servers serverDetailsCollection

	if name == "*" && wildcardOk {
		servers = append(servers, *c...)
	} else {
		if !strings.HasPrefix(name, "*") {
			name = "*" + name
		}

		if !strings.HasSuffix(name, "*") {
			name += "*"
		}

		for _, server := range *c {
			if glob.Glob(strings.ToLower(name), strings.ToLower(server.NameShort)) ||
				strings.EqualFold(server.NameShort, name) {
				servers = append(servers, server)

				break
			}
		}
	}

	return servers
}

func (c *serverDetailsCollection) serverIDsByName(name string, wildcardOk bool) []int {
	var servers []int //nolint:prealloc
	for _, server := range c.byName(name, wildcardOk) {
		servers = append(servers, server.ServerID)
	}

	return servers
}

type rconController struct {
	*rcon.RemoteConsole
	*sync.RWMutex
	attempts           int
	lastConnectSuccess time.Time
	lastConnectAttempt time.Time
}

func (rc *rconController) connected() bool {
	rc.RLock()
	defer rc.RUnlock()

	return rc.RemoteConsole != nil
}

func (rc *rconController) allowedToConnect() bool {
	const (
		waitInterval    = time.Minute
		limitMultiCount = 10
	)

	rc.RLock()
	defer rc.RUnlock()

	if rc.attempts == 0 || rc.RemoteConsole != nil {
		return true
	}

	multi := rc.attempts
	if multi > limitMultiCount {
		multi = limitMultiCount
	}

	return rc.lastConnectAttempt.Add(waitInterval * time.Duration(multi)).Before(time.Now())
}

type serverStateCollector struct {
	log              *zap.Logger
	statusUpdateFreq time.Duration
	msListUpdateFreq time.Duration
	updateTimeout    time.Duration
	masterServerList []serverLocation
	connections      map[int]*rconController
	connectionsMu    *sync.RWMutex
	serverState      map[int]serverDetails
	stateMu          *sync.RWMutex
	configs          []serverConfig
	maxPlayersRx     *regexp.Regexp
}

func newServerStateCollector(logger *zap.Logger) *serverStateCollector {
	const (
		statusUpdateFreq = time.Second * 20
		msListUpdateFreq = time.Minute * 5
		updateTimeout    = time.Second * 5
	)

	return &serverStateCollector{
		log:              logger,
		statusUpdateFreq: statusUpdateFreq,
		msListUpdateFreq: msListUpdateFreq,
		updateTimeout:    updateTimeout,
		connections:      map[int]*rconController{},
		connectionsMu:    &sync.RWMutex{},
		serverState:      map[int]serverDetails{},
		stateMu:          &sync.RWMutex{},
		maxPlayersRx:     regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`),
	}
}

func (c *serverStateCollector) logAddressAdd(logAddress string) {
	time.Sleep(time.Second * 60)

	c.connectionsMu.RLock()
	defer c.connectionsMu.RUnlock()

	for _, server := range c.connections {
		if server.RemoteConsole == nil {
			continue
		}

		_, errExec := server.Exec(fmt.Sprintf("logaddress_add %s", logAddress))
		if errExec != nil {
			c.log.Error("Failed to set logaddress")
		}
	}
}

func (c *serverStateCollector) rcon(serverID int, cmd string) (string, error) {
	c.connectionsMu.RLock()

	conn, found := c.connections[serverID]
	if !found || conn.RemoteConsole == nil {
		c.connectionsMu.RUnlock()

		return "", errUnknownServer
	}

	c.connectionsMu.RUnlock()

	conn.Lock()
	defer conn.Unlock()

	resp, errExec := conn.Exec(cmd)
	if errExec != nil {
		return "", errors.Wrap(errExec, "Failed to perform command")
	}

	return resp, nil
}

func (c *serverStateCollector) broadcast(serverIDs []int, cmd string) map[int]string {
	results := map[int]string{}
	waitGroup := sync.WaitGroup{}

	for _, serverID := range serverIDs {
		waitGroup.Add(1)

		go func(sid int) {
			defer waitGroup.Done()

			resp, errExec := c.rcon(sid, cmd)
			if errExec != nil {
				return
			}

			results[sid] = resp
		}(serverID)
	}

	waitGroup.Wait()

	return results
}

func (c *serverStateCollector) current() serverDetailsCollection {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	var curState []serverDetails //nolint:prealloc
	for _, s := range c.serverState {
		curState = append(curState, s)
	}

	sort.SliceStable(curState, func(i, j int) bool {
		return curState[i].Name < curState[j].Name
	})

	return curState
}

func (c *serverStateCollector) startMSL(ctx context.Context) {
	var (
		log            = c.log.Named("msl_update")
		mlUpdateTicker = time.NewTicker(c.msListUpdateFreq)
	)

	for {
		select {
		case <-mlUpdateTicker.C:
			newMsl, errUpdateMsl := c.updateMSL(ctx)
			if errUpdateMsl != nil {
				log.Error("Failed to update master server list", zap.Error(errUpdateMsl))

				continue
			}

			c.stateMu.Lock()
			c.masterServerList = newMsl
			c.stateMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (c *serverStateCollector) onStatusUpdate(conf serverConfig, newState extra.Status, maxVisible int) {
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

func (c *serverStateCollector) setServerConfigs(configs []serverConfig) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	var gone []serverConfig

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

	for _, config := range configs {
		if _, found := c.serverState[config.ServerID]; !found {
			addr, errResolve := resolveIP(config.Host)
			if errResolve != nil {
				c.log.Warn("Failed to resolve server ip", zap.String("addr", addr), zap.Error(errResolve))
				addr = config.Host
			}

			c.serverState[config.ServerID] = serverDetails{
				ServerID:      config.ServerID,
				Name:          config.DefaultHostname,
				NameShort:     config.Tag,
				Host:          config.Host,
				Port:          config.Port,
				RconPassword:  config.RconPassword,
				ReservedSlots: config.ReservedSlots,
				CC:            config.CC,
				Region:        config.Region,
				Latitude:      config.Latitude,
				Longitude:     config.Longitude,
				IP:            addr,
			}
		}
	}

	c.configs = configs
}

func resolveIP(addr string) (string, error) {
	ipAddr := net.ParseIP(addr)
	if ipAddr != nil {
		return ipAddr.String(), nil
	}

	ips, err := net.LookupIP(addr)
	if err != nil || len(ips) == 0 {
		return "", errors.Wrap(err, "Failed to resolve server dns")
	}

	return ips[0].String(), nil
}

type partialStateUpdate struct {
	Hostname       string `json:"hostname"`
	ShortName      string `json:"short_name"`
	CurrentMap     string `json:"current_map"`
	PlayersReal    int    `json:"players_real"`
	PlayersTotal   int    `json:"players_total"`
	PlayersVisible int    `json:"players_visible"`
}

func (c *serverStateCollector) updateState(serverID int, update partialStateUpdate) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	curState, ok := c.serverState[serverID]
	if !ok {
		return errUnknownServer
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

func (c *serverStateCollector) controllerRcon(controller *rconController, cmd string) (string, error) {
	controller.Lock()
	defer controller.Unlock()

	resp, errRcon := controller.Exec(cmd)

	if errRcon != nil {
		err := errors.Wrapf(errRcon, "Failed to exec rcon command: %s", cmd)
		if errClose := controller.RemoteConsole.Close(); errClose != nil {
			err = gerrors.Join(err, errors.Wrap(errClose, "Failed to close rcon connection"))
		}

		controller.RemoteConsole = nil

		return "", err
	}

	return resp, nil
}

func (c *serverStateCollector) status(controller *rconController) (extra.Status, error) {
	statusResp, errStatus := c.controllerRcon(controller, "status")
	if errStatus != nil {
		return extra.Status{}, errStatus
	}

	status, errParse := extra.ParseStatus(statusResp, true)
	if errParse != nil {
		return extra.Status{}, errors.Wrap(errParse, "Failed to parse status response")
	}

	return status, nil
}

const maxPlayersSupported = 101

func (c *serverStateCollector) maxVisiblePlayers(controller *rconController) (int, error) {
	maxPlayersResp, errMaxPlayers := c.controllerRcon(controller, "sv_visiblemaxplayers")
	if errMaxPlayers != nil {
		return 0, errMaxPlayers
	}

	matches := c.maxPlayersRx.FindStringSubmatch(maxPlayersResp)
	if matches == nil || len(matches) != 2 {
		return 0, errors.New("Failed to parse sv_visiblemaxplayers response")
	}

	maxPlayers, errCast := strconv.ParseInt(matches[1], 10, 32)
	if errCast != nil {
		return 0, errors.Wrap(errCast, "Failed to cast max players value")
	}

	if maxPlayers > maxPlayersSupported {
		maxPlayers = -1
	}

	return int(maxPlayers), nil
}

func (c *serverStateCollector) startStatus(ctx context.Context) {
	var (
		logger             = c.log.Named("statusUpdate")
		statusUpdateTicker = time.NewTicker(c.statusUpdateFreq)
	)

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

				go func(conf serverConfig) {
					defer waitGroup.Done()

					log := logger.Named(conf.Tag)

					c.connectionsMu.Lock()

					controller, found := c.connections[conf.ServerID]
					if !found {
						controller = &rconController{RWMutex: &sync.RWMutex{}, RemoteConsole: nil, attempts: 0, lastConnectAttempt: time.Now()}
						c.connections[conf.ServerID] = controller
					}

					c.connectionsMu.Unlock()

					connected := controller.connected()

					if !connected && !controller.allowedToConnect() {
						return
					}

					if !connected {
						dialCtx, cancel := context.WithTimeout(ctx, time.Second*5)
						newConsole, errDial := rcon.Dial(dialCtx, conf.addr(), conf.RconPassword, c.updateTimeout)

						if errDial != nil {
							log.Debug("Failed to dial rcon", zap.String("err", errDial.Error()))
						}

						cancel()

						controller.Lock()
						controller.lastConnectAttempt = time.Now()

						if newConsole != nil {
							controller.lastConnectSuccess = controller.lastConnectAttempt
							controller.attempts = 0
							controller.RemoteConsole = newConsole
							connected = true
						} else {
							controller.attempts++
						}
						controller.Unlock()
					} else {
						existing.Add(1)
					}

					if !connected {
						return
					}

					status, errStatus := c.status(controller)
					if errStatus != nil {
						return
					}

					maxVisible, errMaxVisible := c.maxVisiblePlayers(controller)
					if errMaxVisible != nil {
						log.Warn("Got invalid max players value", zap.Error(errMaxVisible))
					}

					c.onStatusUpdate(conf, status, maxVisible)

					c.connectionsMu.Lock()
					c.connections[conf.ServerID] = controller
					c.connectionsMu.Unlock()

					successful.Add(1)
				}(serverConfigInstance)
			}

			waitGroup.Wait()

			logger.Debug("RCON update cycle complete",
				zap.Int32("success", successful.Load()),
				zap.Int32("existing", existing.Load()),
				zap.Int32("fail", int32(len(configs))-successful.Load()),
				zap.Duration("duration", time.Since(startTIme)))
		case <-ctx.Done():
			return
		}
	}
}

func (c *serverStateCollector) updateMSL(ctx context.Context) ([]serverLocation, error) {
	allServers, errServers := steamweb.GetServerList(ctx, map[string]string{
		"appid":     "440",
		"dedicated": "1",
	})

	if errServers != nil {
		return nil, errors.Wrap(errServers, "Failed to fetch updated list")
	}

	var ( //nolint:prealloc
		communityServers []serverLocation
		stats            = newGlobalTF2Stats()
	)

	for _, base := range allServers {
		server := serverLocation{
			LatLong: ip2location.LatLong{},
			Server:  base,
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

		mapType := guessMapType(server.Map)

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

func (c *serverStateCollector) start(ctx context.Context) {
	go c.startMSL(ctx)
	go c.startStatus(ctx)
}

func newServerConfig(serverID int, name string, defaultHostname string, address string,
	port int, rconPassword string, reserved int, countryCode string, region string, lat float64, long float64,
) serverConfig {
	return serverConfig{
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

type serverConfig struct {
	ServerID        int
	Tag             string
	DefaultHostname string
	Host            string
	Port            int
	RconPassword    string
	ReservedSlots   int
	CC              string
	Region          string
	Latitude        float64
	Longitude       float64
}

func (config *serverConfig) addr() string {
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

func guessMapType(mapName string) string {
	mapName = strings.TrimPrefix(mapName, "workshop/")
	pieces := strings.SplitN(mapName, "_", 2)

	if len(pieces) == 1 {
		return "unknown"
	}

	return strings.ToLower(pieces[0])
}

type globalTF2StatsSnapshot struct {
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

// func (stats globalTF2StatsSnapshot) trimMapTypes() map[string]int {
//	const minSize = 5
//
//	out := map[string]int{}
//
//	for keyKey, value := range stats.MapTypes {
//		mapKey := keyKey
//		if value < minSize {
//			mapKey = "unknown"
//		}
//
//		out[mapKey] = value
//	}
//
//	return out
// }

func newGlobalTF2Stats() globalTF2StatsSnapshot {
	return globalTF2StatsSnapshot{
		MapTypes:  map[string]int{},
		Regions:   map[string]int{},
		CreatedOn: time.Now(),
	}
}

type serverLocation struct {
	ip2location.LatLong
	steamweb.Server
}
