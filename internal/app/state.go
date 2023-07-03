package app

import (
	"net"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/ryanuber/go-glob"
)

// ServerDetails contains the entire state for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type ServerDetails struct {
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

	Protocol uint8 `json:"protocol"`

	Map string `json:"map"`
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

	Tags    []string       `json:"tags"`
	Players []extra.Player `json:"players"`
}

type BaseServer struct {
	ServerID   int      `json:"server_id"`
	Host       string   `json:"host"`
	Port       int      `json:"port"`
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

type ServerDetailsCollection []ServerDetails

type PlayerServerInfo struct {
	Player   extra.Player
	ServerID int
}

type FindOpts struct {
	Name    string
	IP      *net.IP
	SteamID steamid.SID64
	CIDR    *net.IPNet `json:"cidr"`
}

func (app *App) Find(opts FindOpts) ([]PlayerServerInfo, bool) {
	curState := app.state()
	var found []PlayerServerInfo
	for _, server := range curState {
		for _, player := range server.Players {
			if (opts.SteamID.Valid() && player.SID == opts.SteamID) ||
				(opts.Name != "" && glob.Glob(opts.Name, player.Name)) ||
				(opts.IP != nil && opts.IP.Equal(player.IP)) ||
				(opts.CIDR != nil && opts.CIDR.Contains(player.IP)) {
				found = append(found, PlayerServerInfo{Player: player, ServerID: server.ServerID})
			}
		}
	}

	return found, false
}

func (c ServerDetailsCollection) ByRegion() map[string]ServerDetailsCollection {
	rm := map[string]ServerDetailsCollection{}
	for _, server := range c {
		_, exists := rm[server.Region]
		if !exists {
			rm[server.Region] = ServerDetailsCollection{}
		}

		rm[server.Region] = append(rm[server.Region], server)
	}

	return rm
}

func (c ServerDetailsCollection) ByName(name string) (ServerDetails, bool) {
	for _, server := range c {
		if strings.EqualFold(server.NameShort, name) {
			return server, true
		}
	}

	return ServerDetails{}, false
}
