package state

import (
	"context"
	"errors"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/rumblefrog/go-a2s"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
	"time"
)

var (
	// Current known state of the servers rcon status command
	serverStates     map[int]ServerState
	stateMu          sync.RWMutex
	masterServerList []ServerLocation
	serverConfigs    []*ServerConfig
)

func init() {
	serverStates = map[int]ServerState{}
	masterServerList = []ServerLocation{}
}

type ServerStateCollection []ServerState

func (c ServerStateCollection) ByName(name string, state *ServerState) bool {
	for _, server := range c {
		if strings.EqualFold(server.NameShort, name) {
			*state = server
			return true
		}
	}
	return false
}

func (c ServerStateCollection) ByRegion() map[string][]ServerState {
	rm := map[string][]ServerState{}
	for serverId, server := range c {
		_, exists := rm[server.Region]
		if !exists {
			rm[server.Region] = []ServerState{}
		}
		rm[server.Region] = append(rm[server.Region], c[serverId])
	}
	return rm
}

type FindOpts struct {
	Name    string
	IP      *net.IP
	SteamID steamid.SID64
	CIDR    *net.IPNet `json:"cidr"`
}

func Find(opts FindOpts) (PlayerInfoCollection, bool) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	var found PlayerInfoCollection
	for sid, server := range serverStates {
		for _, player := range server.Players {
			if (opts.SteamID.Valid() && player.SID == opts.SteamID) ||
				(opts.Name != "" && glob.Glob(opts.Name, player.Name)) ||
				(opts.IP != nil && opts.IP.Equal(player.IP)) ||
				(opts.CIDR != nil && opts.CIDR.Contains(player.IP)) {
				found = append(found, PlayerServerInfo{Player: player, ServerId: sid})
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
	ServerId    int       `json:"server_id"`
	NameShort   string    `json:"name_short"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Enabled     bool      `json:"enabled"`
	Region      string    `json:"region"`
	CountryCode string    `json:"cc"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Reserved    int       `json:"reserved"`
	LastUpdate  time.Time `json:"last_update"`
	// A2S
	Protocol uint8  `json:"protocol"`
	Map      string `json:"map"`
	// Name of the folder containing the game files.
	Folder string `json:"folder"`
	// Full name of the game.
	Game string `json:"game"`
	// Steam Application ID of game.
	AppId uint16 `json:"app_id"`
	// Number of players on the server.
	PlayerCount int `json:"player_count"`
	// Maximum number of players the server reports it can hold.
	MaxPlayers int `json:"max_players"`
	// Number of bots on the server.
	Bots int `json:"Bots"`
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
	// The server's 64-bit GameID. If this is present, a more accurate AppID is present in the low 24 bits. The earlier AppID could have been truncated as it was forced into 16-bit storage.
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
	SID           steamid.SID64 `json:"steam_id"`
	ConnectedTime time.Duration `json:"connected_time"`
	State         string        `json:"state"`
	Ping          int           `json:"ping"`
	Loss          int           `json:"-"`
	IP            net.IP        `json:"-"`
	Port          int           `json:"-"`
}

type PlayerServerInfo struct {
	Player   ServerStatePlayer
	ServerId int
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

func State() ServerStateCollection {
	stateMu.RLock()
	defer stateMu.RUnlock()
	var coll ServerStateCollection
	for _, state := range serverStates {
		coll = append(coll, state)
	}
	return coll
}

func updateServers(ctx context.Context) {
	wg := &sync.WaitGroup{}
	results := map[int]ServerState{}
	resultsMu := &sync.RWMutex{}
	for _, config := range serverConfigs {
		wg.Add(1)
		go func(cfg *ServerConfig) {
			defer wg.Done()
			newState, errFetch := cfg.fetch(ctx)
			if errFetch != nil {
				cfg.logger.Error("Failed to update", zap.Error(errFetch))
			}
			resultsMu.Lock()
			results[cfg.ServerId] = newState
			resultsMu.Unlock()
		}(config)
	}
	wg.Wait()
	stateMu.Lock()
	serverStates = results
	stateMu.Unlock()
}

func Start(ctx context.Context, statusUpdateFreq time.Duration, msListUpdateFreq time.Duration, errChan chan error) error {
	statusUpdateTicker := time.NewTicker(statusUpdateFreq)
	mlUpdateTicker := time.NewTicker(msListUpdateFreq)

	for {
		select {
		case <-mlUpdateTicker.C:
			updateMSL(errChan)
		case <-statusUpdateTicker.C:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*20)
			updateServers(localCtx)
			cancel()
		case <-ctx.Done():
			return nil
		}
	}
}

func NewServerConfig(logger *zap.Logger, serverId int, name string, nameShort string, address string, password string, lat float64, long float64, region string, cc string) *ServerConfig {
	return &ServerConfig{
		ServerId:           serverId,
		Name:               name,
		NameShort:          nameShort,
		Address:            address,
		Password:           password,
		Lat:                lat,
		Long:               long,
		rcon:               nil,
		lastRconConnection: time.Time{},
		logger:             logger,
		Region:             region,
		CC:                 cc,
	}
}

type ServerConfig struct {
	ServerId           int
	Name               string
	NameShort          string
	Address            string
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

func (config *ServerConfig) fetch(ctx context.Context) (ServerState, error) {
	var err error
	wg := &sync.WaitGroup{}
	wg.Add(2)
	mu := &sync.RWMutex{}
	stateMu.RLock()
	newState, found := serverStates[config.ServerId]
	stateMu.RUnlock()
	if !found {
		newState = ServerState{
			Region:      config.Region,
			CountryCode: config.CC,
			Latitude:    config.Lat,
			Longitude:   config.Long,
		}
	}

	go func() {
		defer wg.Done()
		if config.a2sConn == nil {
			client, errClient := a2s.NewClient(config.Address, a2s.TimeoutOption(time.Second*10))
			if errClient != nil {
				config.logger.Error("Failed to create a2s client")
				return
			}
			config.a2sConn = client
		}
		result, errQuery := config.a2sConn.QueryInfo()
		if errQuery != nil {
			err = errors.Join(err, errQuery)
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
		newState.AppId = result.ID
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
			newState.SteamID = steamid.SID64(result.ExtendedServerInfo.SteamID)
			newState.GameID = result.ExtendedServerInfo.GameID
			newState.Keywords = strings.Split(result.ExtendedServerInfo.Keywords, ",")
		}
	}()
	go func() {
		defer wg.Done()
		localCtx, cancel := context.WithTimeout(ctx, time.Second*20)
		defer cancel()
		if config.rcon == nil {
			console, errDial := rcon.Dial(localCtx, config.Address, config.Password, time.Second*20)
			if errDial != nil {
				err = errors.Join(err, errDial)
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
				err = errors.Join(err, errRcon)
				return
			}
			status, errParse := extra.ParseStatus(resp, true)
			if errParse != nil {
				err = errors.Join(err, errParse)
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
					SID:           p.SID,
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

func SetServers(configs []*ServerConfig) {
	stateMu.Lock()
	defer stateMu.Unlock()
	serverConfigs = configs
}

func MasterServerList() []ServerLocation {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return masterServerList
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

func SteamRegionIdString(region SvRegion) string {
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
func updateMSL(errChan chan error) {
	allServers, errServers := steamweb.GetServerList(map[string]string{
		"appid":     "440",
		"dedicated": "1",
	})
	if errServers != nil {
		errChan <- errServers
		return
	}
	var communityServers []ServerLocation
	stats := NewGlobalTF2Stats()
	for _, baseServer := range allServers {
		server := ServerLocation{
			LatLong: ip2location.LatLong{},
			Server:  baseServer,
		}
		stats.ServersTotal++
		stats.Players += server.Players
		stats.Bots += server.Bots
		if server.MaxPlayers > 0 && server.Players >= server.MaxPlayers {
			stats.CapacityFull++
		} else if server.Players == 0 {
			stats.CapacityEmpty++
		} else {
			stats.CapacityPartial++
		}
		if server.Secure {
			stats.Secure++
		}
		region := SteamRegionIdString(SvRegion(server.Region))
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
		if strings.Contains(server.Gametype, "valve") ||
			!server.Dedicated ||
			!server.Secure {
			stats.ServersCommunity++
			continue
		}
		communityServers = append(communityServers, server)
	}
	stateMu.Lock()
	masterServerList = communityServers
	stateMu.Unlock()
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
	StatId           int64          `json:"stat_id"`
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
