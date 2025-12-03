package servers

import (
	"errors"
	"net"
	"time"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrPlayerNotFound = errors.New("could not find player")
	ErrUnknownServer  = errors.New("unknown server")
)

type LogFilePayload struct {
	ServerID   int
	ServerName string
	Lines      []string
	Map        string
}

type PartialStateUpdate struct {
	Hostname       string `json:"hostname"`
	ShortName      string `json:"short_name"`
	CurrentMap     string `json:"current_map"`
	PlayersReal    int    `json:"players_real"`
	PlayersTotal   int    `json:"players_total"`
	PlayersVisible int    `json:"players_visible"`
}

type Status struct {
	extra.Status

	Humans int
	Bots   int
}

type state struct {
	// IP is a distinct entry vs host since steam no longer allows steam:// protocol handler links to use a fqdn
	IP         string `json:"ip"`
	Port       uint16 `json:"port"`
	IPPublic   string `json:"ip_public"`
	PortPublic uint16 `json:"port_public"`

	LastUpdate    time.Time `json:"last_update"`
	ReservedSlots int       `json:"reserved_slots"`
	Protocol      uint8     `json:"protocol"`
	Map           string    `json:"map"`
	// Name of the folder containing the game files.
	Folder   string `json:"folder"`
	ServerOS string `json:"server_os"`
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
	Tags    []string  `json:"tags"`
	Players []*Player `json:"players"`
	// How many human players in the server
	Humans int `json:"humans"`
	// HasSynchronizedDNS tracks if the server has done its initial DNS update cycle. This is required
	// for future change detection and updates.
	HasSynchronizedDNS bool              `json:"has_synchronized_dns"`
	Rules              map[string]string `json:"rules"`
}

type Player struct {
	UserID        int
	Name          string
	SID           steamid.SteamID
	ConnectedTime time.Duration
	Ping          int
	Loss          int
	State         string
	IP            net.IP
	Port          int
	Score         int
}
