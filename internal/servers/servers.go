package servers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrUnknownServer = errors.New("unknown server")
	ErrGetServer     = errors.New("failed to get server")
)

type Query struct {
	ServerID        int    `query:"server_id"`
	IncludeDisabled bool   `query:"include_disabled"`
	SDROnly         bool   `query:"sdr_only"`
	ShortName       string `query:"short_name"`
	Password        string `query:"password"`
	IncludeDeleted  bool
}

type ServerInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

type ServerPermission struct {
	SteamID         steamid.SID          `json:"steam_id"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	Flags           string               `json:"flags"`
}

func NewServer(shortName string, address string, port uint16) Server {
	return Server{
		ShortName:      shortName,
		Address:        address,
		Port:           port,
		RCON:           stringutil.SecureRandomString(10),
		ReservedSlots:  0,
		Password:       stringutil.SecureRandomString(10),
		IsEnabled:      true,
		EnableStats:    true,
		TokenCreatedOn: time.Unix(0, 0),
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type Server struct {
	// Auto generated id
	ServerID int `json:"server_id"`
	// ShortName is a short reference name for the server eg: us-1
	// This is used as a unique identifier for servers and is used for many different things such as paths,
	// so it's best to keep it short and without whitespace.
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
	// Address is the ip of the server
	Address string `json:"address"`
	// Internal/VPN network. When defined it's used for things like pulling demos over ssh.
	AddressInternal string `json:"address_internal"`
	SDREnabled      bool   `json:"sdr_enabled"`
	// Port is the port of the server
	Port uint16 `json:"port"`
	// RCON is the RCON password for the server
	RCON          string `json:"rcon"`
	ReservedSlots int    `json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls (permanent Refresh token)
	Password    string  `json:"password"`
	IsEnabled   bool    `json:"is_enabled"`
	Deleted     bool    `json:"deleted"`
	Region      string  `json:"region"`
	CC          string  `json:"cc"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	LogSecret   int     `json:"log_secret"`
	EnableStats bool    `json:"enable_stats"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn     time.Time `json:"token_created_on"`
	CreatedOn          time.Time `json:"created_on"`
	UpdatedOn          time.Time `json:"updated_on"`
	DiscordSeedRoleIDs []string  `json:"discord_seed_role_ids"` //nolint:tagliatelle
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) AddrInternalOrDefault() string {
	if s.AddressInternal != "" {
		return s.AddressInternal
	}

	return s.Address
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

// SafeServer provides a server struct stripped of any sensitive info suitable for public-facing
// services.
type SafeServer struct {
	ServerID   int      `json:"server_id"`
	Host       string   `json:"host"`
	Port       uint16   `json:"port"`
	IP         string   `json:"ip"`
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
	Humans     int      `json:"humans"`
}

func NewServers(repository Repository) Servers {
	return Servers{repository: repository}
}

type Servers struct {
	repository Repository
}

// Delete performs a soft delete of the server. We use soft deleted because we dont wand to delete all the relationships
// that rely on this suchs a stats.
func (s *Servers) Delete(ctx context.Context, serverID int) error {
	if serverID <= 0 {
		return httphelper.ErrInvalidParameter
	}

	server, errServer := s.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	server.Deleted = true

	if err := s.repository.Save(ctx, &server); err != nil {
		return err
	}

	return nil
}

func (s *Servers) Server(ctx context.Context, serverID int) (Server, error) {
	if serverID <= 0 {
		return Server{}, ErrNotFound
	}

	servers, err := s.repository.Query(ctx, Query{ServerID: serverID, IncludeDisabled: true})
	if err != nil {
		return Server{}, err
	}

	if len(servers) != 1 {
		return Server{}, ErrNotFound
	}

	return servers[0], nil
}

func (s *Servers) ServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	return s.repository.GetServerPermissions(ctx)
}

func (s *Servers) Servers(ctx context.Context, filter Query) ([]Server, error) {
	return s.repository.Query(ctx, filter)
}

func (s *Servers) GetByName(ctx context.Context, serverName string) (Server, error) {
	server, errServer := s.repository.Query(ctx, Query{ShortName: serverName})

	if errServer != nil {
		return Server{}, errServer
	}

	if len(server) == 0 {
		return Server{}, ErrUnknownServer
	}

	return server[0], nil
}

func (s *Servers) GetByPassword(ctx context.Context, serverPassword string) (Server, error) {
	server, errServer := s.repository.Query(ctx, Query{Password: serverPassword})

	if errServer != nil {
		return Server{}, errServer
	}

	if len(server) == 0 {
		return Server{}, ErrUnknownServer
	}

	return server[0], nil
}

func (s *Servers) Save(ctx context.Context, server Server) (Server, error) {
	if server.ServerID > 0 {
		server.UpdatedOn = time.Now()
	}

	if err := s.repository.Save(ctx, &server); err != nil {
		return Server{}, err
	}

	return s.Server(ctx, server.ServerID)
}
