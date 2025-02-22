package domain

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type StateUsecase interface {
	Start(ctx context.Context) error
	Broadcast(ctx context.Context, serverIDs []int, cmd string) map[int]string
	ByName(name string, wildcardOk bool) []ServerState
	ByServerID(serverID int) (ServerState, bool)
	CSay(ctx context.Context, serverID int, message string) error
	Current() []ServerState
	ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error)
	ExecServer(ctx context.Context, serverID int, cmd string) (string, error)
	Find(name string, steamID steamid.SteamID, addr net.IP, cidr *net.IPNet) []PlayerServerInfo
	FindByCIDR(cidr *net.IPNet) []PlayerServerInfo
	FindByIP(addr net.IP) []PlayerServerInfo
	FindByName(name string) []PlayerServerInfo
	FindBySteamID(steamID steamid.SteamID) []PlayerServerInfo
	Kick(ctx context.Context, target steamid.SteamID, reason Reason) error
	KickPlayerID(ctx context.Context, targetPlayerID int, targetServerID int, reason Reason) error
	LogAddressAdd(ctx context.Context, logAddress string)
	LogAddressDel(ctx context.Context, logAddress string)
	OnFindExec(ctx context.Context, name string, steamID steamid.SteamID, ip net.IP, cidr *net.IPNet, onFoundCmd func(info PlayerServerInfo) string) error
	PSay(ctx context.Context, target steamid.SteamID, message string) error
	Say(ctx context.Context, serverID int, message string) error
	ServerIDsByName(name string, wildcardOk bool) []int
	Silence(ctx context.Context, target steamid.SteamID, reason Reason) error
	SortRegion() map[string][]ServerState
	Update(serverID int, update PartialStateUpdate) error
}

type StateRepository interface {
	Start(ctx context.Context)
	GetServer(serverID int) (ServerConfig, error)
	Update(serverID int, update PartialStateUpdate) error
	Current() []ServerState
	Configs() []ServerConfig
	ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error)
}

type LogFilePayload struct {
	ServerID   int
	ServerName string
	Lines      []string
	Map        string
}

type ServerConfig struct {
	ServerID        int
	Tag             string
	DefaultHostname string
	Host            string
	Port            int
	Enabled         bool
	RconPassword    string
	ReservedSlots   int
	CC              string
	Region          string
	Latitude        float64
	Longitude       float64
}

func (config *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

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
	SteamID steamid.SteamID `json:"steam_id"`
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
	Humans  int            `json:"humans"`
}
