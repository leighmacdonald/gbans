package state

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/rumblefrog/go-a2s"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
)

type ServerStateCollector struct {
	sync.RWMutex
	serverStates     map[int]*ServerState
	masterServerList []ServerLocation
	serverConfigs    []*ServerConfig
}

func NewServerStateCollector() *ServerStateCollector {
	return &ServerStateCollector{
		serverStates:     make(map[int]*ServerState),
		masterServerList: nil,
		serverConfigs:    nil,
	}
}

func (c *ServerStateCollector) ByName(name string, state *ServerState) bool {
	for _, server := range c.serverStates {
		if strings.EqualFold(server.NameShort, name) {
			*state = *server

			return true
		}
	}

	return false
}

func (c *ServerStateCollector) ByRegion() map[string][]ServerState {
	rm := map[string][]ServerState{}
	for _, server := range c.serverStates {
		_, exists := rm[server.Region]
		if !exists {
			rm[server.Region] = []ServerState{}
		}

		rm[server.Region] = append(rm[server.Region], *server)
	}

	return rm
}

type FindOpts struct {
	Name    string
	IP      *net.IP
	SteamID steamid.SID64
	CIDR    *net.IPNet `json:"cidr"`
}

func (c *ServerStateCollector) Find(opts FindOpts) (PlayerInfoCollection, bool) {
	c.RLock()
	defer c.RUnlock()
	var found PlayerInfoCollection
	for sid, server := range c.serverStates {
		for _, player := range server.Players {
			if (opts.SteamID.Valid() && player.SteamID == opts.SteamID) ||
				(opts.Name != "" && glob.Glob(opts.Name, player.Name)) ||
				(opts.IP != nil && opts.IP.Equal(player.IP)) ||
				(opts.CIDR != nil && opts.CIDR.Contains(player.IP)) {
				found = append(found, PlayerServerInfo{Player: player, ServerID: sid})
			}
		}
	}

	return found, false
}

// ServerState contains the entire state for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type ServerState struct {
	// Database
	ServerID   int       `json:"server_id"`
	NameShort  string    `json:"name_short"`
	Name       string    `json:"name"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	Enabled    bool      `json:"enabled"`
	Region     string    `json:"region"`
	CC         string    `json:"cc"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Reserved   int       `json:"reserved"`
	LastUpdate time.Time `json:"last_update"`
	// A2S
	Protocol uint8  `json:"protocol"`
	Map      string `json:"map"`
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
	// Indicates the type of server
	// Rag Doll Kung Fu servers always return 0 for "Server type."
	ServerType string `json:"server_type"`
	// Indicates the operating system of the server
	ServerOS string `json:"server_os"`
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

	// RCON Sourced
	Players []ServerStatePlayer `json:"players"`
}

type ServerStatePlayer struct {
	UserID        int           `json:"user_id"`
	Name          string        `json:"name"`
	SteamID       steamid.SID64 `json:"steam_id"`
	ConnectedTime time.Duration `json:"connected_time"`
	State         string        `json:"state"`
	Ping          int           `json:"ping"`
	Loss          int           `json:"-"`
	IP            net.IP        `json:"-"`
	Port          int           `json:"-"`
}

type PlayerServerInfo struct {
	Player   ServerStatePlayer
	ServerID int
}

type PlayerInfoCollection []PlayerServerInfo

func (pic PlayerInfoCollection) NotEmpty() bool {
	return len(pic) > 0
}

func NewPlayerInfo() PlayerServerInfo {
	return PlayerServerInfo{
		Player: ServerStatePlayer{},
	}
}

func (c *ServerStateCollector) State() []ServerState {
	c.RLock()
	defer c.RUnlock()

	coll := make([]ServerState, len(c.serverStates))
	for index, state := range c.serverStates {
		coll[index] = *state
	}

	return coll
}

func (c *ServerStateCollector) updateServers(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}

	for _, config := range c.serverConfigs {
		waitGroup.Add(1)
		go func(cfg *ServerConfig) {
			defer waitGroup.Done()
			newState, errFetch := c.fetch(ctx, cfg)
			if errFetch != nil {
				cfg.logger.Error("Failed to update", zap.Error(errFetch))
			}
			// TODO update not overwrite
			c.serverStates[cfg.ServerID] = newState
		}(config)
	}

	waitGroup.Wait()
}

func (c *ServerStateCollector) Start(ctx context.Context, statusUpdateFreq time.Duration, msListUpdateFreq time.Duration, errChan chan error) error {
	statusUpdateTicker := time.NewTicker(statusUpdateFreq)
	mlUpdateTicker := time.NewTicker(msListUpdateFreq)

	for {
		select {
		case <-mlUpdateTicker.C:
			c.updateMSL(ctx, errChan)
		case <-statusUpdateTicker.C:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*20)
			c.updateServers(localCtx)
			cancel()
		case <-ctx.Done():
			return nil
		}
	}
}

func NewServerConfig(logger *zap.Logger, serverID int, name string, nameShort string, address string, port int, password string, lat float64, long float64, region string, cc string) *ServerConfig {
	return &ServerConfig{
		ServerID:           serverID,
		Name:               name,
		NameShort:          nameShort,
		Host:               address,
		Port:               port,
		Password:           password,
		Lat:                lat,
		Long:               long,
		Region:             region,
		CC:                 cc,
		rcon:               nil,
		a2sConn:            nil,
		lastRconConnection: time.Time{},
		logger:             logger,
	}
}

type ServerConfig struct {
	ServerID           int
	Name               string
	NameShort          string
	Host               string
	Port               int
	Password           string
	Lat                float64
	Long               float64
	Region             string
	CC                 string
	rcon               *rcon.RemoteConsole
	a2sConn            *a2s.Client
	lastRconConnection time.Time
	logger             *zap.Logger
}

func (config *ServerConfig) addr() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

func (c *ServerStateCollector) fetch(ctx context.Context, config *ServerConfig) (*ServerState, error) {
	var err error
	wg := &sync.WaitGroup{}
	wg.Add(2)
	mu := &sync.RWMutex{}
	errMu := sync.RWMutex{}
	c.RLock()
	newState, found := c.serverStates[config.ServerID]
	c.RUnlock()
	if !found {
		newState = &ServerState{
			ServerID:  config.ServerID,
			NameShort: config.NameShort,
			Name:      config.Name,
			Host:      config.Host,
			Port:      config.Port,
			Enabled:   true,
			Region:    config.Region,
			CC:        config.CC,
			Latitude:  config.Lat,
			Longitude: config.Long,
		}
	}

	go func() {
		defer wg.Done()
		if config.a2sConn == nil {
			client, errClient := a2s.NewClient(config.addr(), a2s.TimeoutOption(time.Second*10))
			if errClient != nil {
				config.logger.Error("Failed to create a2s client")

				return
			}
			config.a2sConn = client
		}
		result, errQuery := config.a2sConn.QueryInfo()
		if errQuery != nil {
			errMu.Lock()
			err = errors.Join(err, errQuery)
			errMu.Unlock()
			if config.a2sConn != nil {
				_ = config.a2sConn.Close()
				config.a2sConn = nil
			}

			return
		}
		mu.Lock()
		defer mu.Unlock()
		newState.Protocol = result.Protocol
		newState.Name = result.Name
		newState.Map = result.Map
		newState.Folder = result.Folder
		newState.Game = result.Game
		newState.AppID = result.ID
		newState.PlayerCount = int(result.Players)
		newState.MaxPlayers = int(result.MaxPlayers)
		newState.Bots = int(result.Bots)
		newState.ServerType = result.ServerType.String()
		newState.ServerOS = result.ServerOS.String()
		newState.Password = !result.Visibility
		newState.VAC = result.VAC
		newState.Version = result.Version
		if result.SourceTV != nil {
			newState.STVPort = result.SourceTV.Port
			newState.STVName = result.SourceTV.Name
		}
		if result.ExtendedServerInfo != nil {
			newState.SteamID = steamid.New(result.ExtendedServerInfo.SteamID)
			newState.GameID = result.ExtendedServerInfo.GameID
			newState.Keywords = strings.Split(result.ExtendedServerInfo.Keywords, ",")
		}
	}()
	go func() {
		defer wg.Done()
		localCtx, cancel := context.WithTimeout(ctx, time.Second*20)
		defer cancel()
		if config.rcon == nil {
			console, errDial := rcon.Dial(localCtx, config.addr(), config.Password, time.Second*20)
			if errDial != nil {
				errMu.Lock()
				err = errors.Join(err, errDial)
				errMu.Unlock()

				return
			}
			config.rcon = console
		}
		if config.rcon != nil {
			resp, errRcon := config.rcon.Exec("status")
			if errRcon != nil {
				if config.rcon != nil {
					_ = config.rcon.Close()
					config.rcon = nil
				}
				errMu.Lock()
				err = errors.Join(err, errRcon)
				errMu.Unlock()

				return
			}
			status, errParse := extra.ParseStatus(resp, true)
			if errParse != nil {
				errMu.Lock()
				err = errors.Join(err, errParse)
				errMu.Unlock()

				return
			}
			mu.Lock()
			defer mu.Unlock()
			newState.Map = status.Map
			newState.Edicts = status.Edicts
			newState.Keywords = status.Tags
			newState.Version = status.Version
			newState.Name = status.ServerName
			var players []ServerStatePlayer
			for _, p := range status.Players {
				players = append(players, ServerStatePlayer{
					UserID:        p.UserID,
					Name:          p.Name,
					SteamID:       p.SID,
					ConnectedTime: p.ConnectedTime,
					State:         p.State,
					Ping:          p.Ping,
					Loss:          p.Loss,
					IP:            p.IP,
					Port:          p.Port,
				})
			}
			newState.Players = players
		}
	}()
	wg.Wait()

	return newState, err
}

func (c *ServerStateCollector) SetServers(configs []*ServerConfig) {
	c.Lock()
	defer c.Unlock()

	c.serverConfigs = configs
}

func (c *ServerStateCollector) MasterServerList() []ServerLocation {
	c.RLock()
	defer c.RUnlock()

	return c.masterServerList
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

func (c *ServerStateCollector) updateMSL(ctx context.Context, errChan chan error) {
	allServers, errServers := steamweb.GetServerList(ctx, map[string]string{
		"appid":     "440",
		"dedicated": "1",
	})
	if errServers != nil {
		errChan <- errServers

		return
	}
	var communityServers []ServerLocation //nolint:prealloc
	stats := NewGlobalTF2Stats()
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
	c.Lock()
	c.masterServerList = communityServers
	c.Unlock()
}

func GuessMapType(mapName string) string {
	mapName = strings.TrimPrefix(mapName, "workshop/")
	pieces := strings.SplitN(mapName, "_", 2)
	if len(pieces) == 1 {
		return "unknown"
	} else {
		return strings.ToLower(pieces[0])
	}
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
	for k, v := range stats.MapTypes {
		mapKey := k
		if v < minSize {
			mapKey = "unknown"
		}
		out[mapKey] = v
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
