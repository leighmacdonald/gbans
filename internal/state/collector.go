package state

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
)

var (
	ErrRCONCommand      = errors.New("failed to execute rcon command")
	ErrFailedToDialRCON = errors.New("failed to connect to conf")
)

// ServerState contains the entire State for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type ServerState struct {
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
	// ServerStore's SteamID.
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

type Collector struct {
	log              *zap.Logger
	statusUpdateFreq time.Duration
	msListUpdateFreq time.Duration
	updateTimeout    time.Duration
	masterServerList []serverLocation
	serverState      map[int]ServerState
	stateMu          *sync.RWMutex
	configs          []serverConfig
	maxPlayersRx     *regexp.Regexp
}

func NewCollector(logger *zap.Logger) *Collector {
	const (
		statusUpdateFreq = time.Second * 20
		msListUpdateFreq = time.Minute * 5
		updateTimeout    = time.Second * 5
	)

	return &Collector{
		log:              logger,
		statusUpdateFreq: statusUpdateFreq,
		msListUpdateFreq: msListUpdateFreq,
		updateTimeout:    updateTimeout,
		serverState:      map[int]ServerState{},
		stateMu:          &sync.RWMutex{},
		maxPlayersRx:     regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`),
	}
}

func (c *Collector) Find(name string, steamID steamid.SID64, addr net.IP, cidr *net.IPNet) []model.PlayerServerInfo {
	var found []model.PlayerServerInfo

	for server := range c.serverState {
		for _, player := range c.serverState[server].Players {
			matched := false
			if steamID.Valid() && player.SID == steamID {
				matched = true
			}

			if name != "" {
				queryName := name
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

			if addr != nil && addr.Equal(player.IP) {
				matched = true
			}

			if cidr != nil && cidr.Contains(player.IP) {
				matched = true
			}

			if matched {
				found = append(found, model.PlayerServerInfo{Player: player, ServerID: c.serverState[server].ServerID})
			}
		}
	}

	return found
}

func (c *Collector) SortRegion() map[string][]ServerState {
	serverMap := map[string][]ServerState{}
	for _, server := range c.serverState {
		_, exists := serverMap[server.Region]
		if !exists {
			serverMap[server.Region] = []ServerState{}
		}

		serverMap[server.Region] = append(serverMap[server.Region], server)
	}

	return serverMap
}

func (c *Collector) ByServerID(serverID int) (ServerState, bool) {
	for _, server := range c.serverState {
		if server.ServerID == serverID {
			return server, true
		}
	}

	return ServerState{}, false
}

func (c *Collector) ByName(name string, wildcardOk bool) []ServerState {
	var servers []ServerState

	if name == "*" && wildcardOk {
		for _, server := range c.serverState {
			servers = append(servers, server)
		}
	} else {
		if !strings.HasPrefix(name, "*") {
			name = "*" + name
		}

		if !strings.HasSuffix(name, "*") {
			name += "*"
		}

		for _, server := range c.serverState {
			if glob.Glob(strings.ToLower(name), strings.ToLower(server.NameShort)) ||
				strings.EqualFold(server.NameShort, name) {
				servers = append(servers, server)

				break
			}
		}
	}

	return servers
}

func (c *Collector) ServerIDsByName(name string, wildcardOk bool) []int {
	var servers []int //nolint:prealloc
	for _, server := range c.ByName(name, wildcardOk) {
		servers = append(servers, server.ServerID)
	}

	return servers
}

func (c *Collector) logAddressAdd(ctx context.Context, logAddress string) {
	c.Broadcast(ctx, nil, fmt.Sprintf("logaddress_add %s", logAddress))
}

// OnFindExec is a helper function used to execute rcon commands against any players found in the query.
func (c *Collector) OnFindExec(ctx context.Context, name string, steamID steamid.SID64,
	ip net.IP, cidr *net.IPNet, onFoundCmd func(info model.PlayerServerInfo) string,
) error {
	currentState := c.Current()
	players := c.Find(name, steamID, ip, cidr)

	if len(players) == 0 {
		return errs.ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		for _, server := range currentState {
			if player.ServerID == server.ServerID {
				_, errRcon := c.ExecServer(ctx, server.ServerID, onFoundCmd(player))
				if errRcon != nil {
					err = errors.Join(errRcon)
				}
			}
		}
	}

	return err
}

var ErrUnknownServerID = errors.New("unknown server id")

func (c *Collector) ExecServer(ctx context.Context, serverID int, cmd string) (string, error) {
	var conf serverConfig

	for _, server := range c.configs {
		if server.ServerID == serverID {
			conf = server

			break
		}
	}

	if conf.ServerID == 0 {
		return "", ErrUnknownServerID
	}

	return c.ExecRaw(ctx, conf.addr(), conf.RconPassword, cmd)
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
		c.log.Error("Could not close rcon connection", zap.Error(errClose))
	}

	return resp, nil
}

func (c *Collector) Broadcast(ctx context.Context, serverIDs []int, cmd string) map[int]string {
	results := map[int]string{}
	waitGroup := sync.WaitGroup{}

	if len(serverIDs) == 0 {
		for _, conf := range c.configs {
			serverIDs = append(serverIDs, conf.ServerID)
		}
	}

	for _, serverID := range serverIDs {
		waitGroup.Add(1)

		go func(sid int) {
			defer waitGroup.Done()

			resp, errExec := c.ExecServer(ctx, sid, cmd)
			if errExec != nil {
				c.log.Error("Failed to exec server command", zap.Int("server_id", sid), zap.Error(errExec))

				return
			}

			results[sid] = resp
		}(serverID)
	}

	waitGroup.Wait()

	return results
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

func (c *Collector) startMSL(ctx context.Context) {
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

func (c *Collector) onStatusUpdate(conf serverConfig, newState extra.Status, maxVisible int) {
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

func (c *Collector) setServerConfigs(configs []serverConfig) {
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

	for _, cfg := range configs {
		if _, found := c.serverState[cfg.ServerID]; !found {
			addr, errResolve := ResolveIP(cfg.Host)
			if errResolve != nil {
				c.log.Warn("Failed to resolve server ip", zap.String("addr", addr), zap.Error(errResolve))
				addr = cfg.Host
			}

			c.serverState[cfg.ServerID] = ServerState{
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

func (c *Collector) Update(serverID int, update model.PartialStateUpdate) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	curState, ok := c.serverState[serverID]
	if !ok {
		return errs.ErrUnknownServer
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
	ErrServerListRequest = errors.New("failed to fetch updated list")
	ErrDNSResolve        = errors.New("failed to resolve server dns")
	ErrRCONExecCommand   = errors.New("failed to perform command")
)

func (c *Collector) status(ctx context.Context, serverID int) (extra.Status, error) {
	statusResp, errStatus := c.ExecServer(ctx, serverID, "status")
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
	maxPlayersResp, errMaxPlayers := c.ExecServer(ctx, serverID, "sv_visiblemaxplayers")
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

					status, errStatus := c.status(ctx, conf.ServerID)
					if errStatus != nil {
						return
					}

					maxVisible, errMaxVisible := c.maxVisiblePlayers(ctx, conf.ServerID)
					if errMaxVisible != nil {
						log.Warn("Got invalid max players value", zap.Error(errMaxVisible))
					}

					c.onStatusUpdate(conf, status, maxVisible)

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

func (c *Collector) updateMSL(ctx context.Context) ([]serverLocation, error) {
	allServers, errServers := steamweb.GetServerList(ctx, map[string]string{
		"appid":     "440",
		"dedicated": "1",
	})

	if errServers != nil {
		return nil, errors.Join(errServers, ErrServerListRequest)
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

type ServerStore interface {
	GetServers(ctx context.Context, filter model.ServerQueryFilter) ([]model.Server, int64, error)
}

func (c *Collector) Start(ctx context.Context, configFunc func() config.Config, database func() ServerStore) {
	var (
		log          = c.log.Named("State")
		trigger      = make(chan any)
		updateTicker = time.NewTicker(time.Minute * 30)
	)

	go c.startMSL(ctx)
	go c.startStatus(ctx)

	go func() {
		trigger <- true
	}()

	for {
		select {
		case <-updateTicker.C:
			trigger <- true
		case <-trigger:
			servers, _, errServers := database().GetServers(ctx, model.ServerQueryFilter{
				QueryFilter:     model.QueryFilter{Deleted: false},
				IncludeDisabled: false,
			})
			if errServers != nil && !errors.Is(errServers, errs.ErrNoResult) {
				log.Error("Failed to fetch servers, cannot update State", zap.Error(errServers))

				continue
			}

			var configs []serverConfig
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

			conf := configFunc()
			if conf.Debug.AddRCONLogAddress != "" {
				c.logAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
			}

		case <-ctx.Done():
			return
		}
	}
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
