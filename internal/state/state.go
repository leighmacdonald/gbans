package state

import (
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net"
	"strings"
	"time"
)

// TODO move findPlayerBy* methods to here
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

// ServerState contains the entire state for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type ServerState struct {
	// Database
	ServerId    int       `json:"server_id"`
	Name        string    `json:"name"`
	NameShort   string    `json:"name_short"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Enabled     bool      `json:"enabled"`
	Region      string    `json:"region"`
	CountryCode string    `json:"cc"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	Reserved    int       `json:"reserved"`
	LastUpdate  time.Time `json:"last_update"`
	// A2S
	NameA2S  string `json:"name_a2s"` // The live name can differ from default
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

type PlayerInfo struct {
	Player  *ServerStatePlayer
	Server  *store.Server
	SteamID steamid.SID64
	InGame  bool
	Valid   bool
}

func NewPlayerInfo() PlayerInfo {
	return PlayerInfo{
		Player:  &ServerStatePlayer{},
		Server:  nil,
		SteamID: 0,
		InGame:  false,
		Valid:   false,
	}
}
