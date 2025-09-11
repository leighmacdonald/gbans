package state

import (
	"fmt"
	"time"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

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
	Port            uint16
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
	// Internal unique ID of the server
	ServerID int `json:"server_id"`
	// A shorthand unique identifier for the server.
	NameShort string `json:"name_short"`
	// A full hostname for the server. This is just the default value and will be
	// updated dynamically from RCON.
	Name string `json:"name"`
	Host string `json:"host"`
	// IP is a distinct entry vs host since steam no longer allows steam:// protocol handler links to use a fqdn
	IP         string `json:"ip"`
	Port       uint16 `json:"port"`
	IPPublic   string `json:"ip_public"`
	PortPublic uint16 `json:"port_public"`

	Enabled       bool      `json:"enabled"`
	Region        string    `json:"region"`
	CC            string    `json:"cc"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
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
	// STVIP is the public ip of the stv server
	STVIP string `json:"stvip"`
	// Spectator port number for SourceTV.
	STVPort uint16 `json:"stv_port"`
	// Name of the spectator server for SourceTV.
	STVName string `json:"stv_name"`
	// A collection of the comma delimited values of sv_tags
	Tags    []string       `json:"tags"`
	Players []extra.Player `json:"players"`
	// How many human players in the server
	Humans int `json:"humans"`
	// HasSynchronizedDNS tracks if the server has done its initial DNS update cycle. This is required
	// for future change detection and updates.
	HasSynchronizedDNS bool `json:"has_synchronized_dns"`
}
