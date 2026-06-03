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
	Hostname       string
	ShortName      string
	CurrentMap     string
	PlayersReal    int
	PlayersTotal   int
	PlayersVisible int
}

type Status struct {
	extra.Status

	Humans int
	Bots   int
}

type state struct {
	// IP is a distinct entry vs host since steam no longer allows steam:// protocol handler links to use a fqdn
	IP         string
	Port       uint16
	IPPublic   string
	PortPublic uint16

	LastUpdate    time.Time
	ReservedSlots int
	Protocol      uint8
	Hostname      string
	Map           string
	// Name of the folder containing the game files.
	Folder   string
	ServerOS string
	// Full name of the game.
	Game string
	// Steam Application ID of game.
	AppID uint16
	// Number of players on the server.
	PlayerCount int32
	// Maximum number of players the server reports it can hold.
	MaxPlayers int32
	// Maximum number of players the server reports it can hold, visible to the public.
	MaxPlayersVisible int32
	// Number of bots on the server.
	Bots int32
	// Indicates whether the server requires a password
	Password bool
	// Specifies whether the server uses VAC
	VAC bool
	// Version of the game installed on the server.
	Version string
	// ServerStore's SteamID.
	SteamID steamid.SteamID
	// Tags that describe the game according to the server (for future use.)
	Keywords []string
	Edicts   []int
	// The server's 64-bit GameID. If this is present, a more accurate AppID is present in the low 24 bits.
	// The earlier AppID could have been truncated as it was forced into 16-bit storage.
	GameID uint64 // Needed?
	// STVIP is the public ip of the stv server
	STVIP string
	// Spectator port number for SourceTV.
	STVPort uint16
	// Name of the spectator server for SourceTV.
	STVName string
	// A collection of the comma delimited values of sv_tags
	Tags    []string
	Players []*Player
	// How many human players in the server
	Humans int32
	// HasSynchronizedDNS tracks if the server has done its initial DNS update cycle. This is required
	// for future change detection and updates.
	HasSynchronizedDNS bool
	Rules              map[string]string
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
