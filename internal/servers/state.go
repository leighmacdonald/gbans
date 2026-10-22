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
	Hostname      string
	Map           string
	// Number of players on the server.
	PlayerCount int32
	// Maximum number of players the server reports it can hold.
	MaxPlayers int32
	// Maximum number of players the server reports it can hold, visible to the public.
	MaxPlayersVisible int32
	// Number of bots on the server.
	Bots int32
	// Version of the game installed on the server.
	Version string
	Edicts  []int
	// STVIP is the public ip of the stv server
	STVIP string
	// Spectator port number for SourceTV.
	STVPort uint16
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
}
