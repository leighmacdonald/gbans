package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrUnknownServerID = errors.New("unknown server id")
	ErrUnknownServer   = errors.New("unknown server")
)

type RequestServerUpdate struct {
	ServerID        int     `json:"server_id"`
	ServerName      string  `json:"server_name"`
	ServerNameShort string  `json:"server_name_short"`
	Host            string  `json:"host"`
	Port            uint16  `json:"port"`
	ReservedSlots   int     `json:"reserved_slots"`
	Password        string  `json:"password"`
	RCON            string  `json:"rcon"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	CC              string  `json:"cc"`
	DefaultMap      string  `json:"default_map"`
	Region          string  `json:"region"`
	IsEnabled       bool    `json:"is_enabled"`
	EnableStats     bool    `json:"enable_stats"`
	LogSecret       int     `json:"log_secret"`
	AddressInternal string  `json:"address_internal"`
	SDREnabled      bool    `json:"sdr_enabled"`
}

type ServerInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

var ErrResolveIP = errors.New("failed to resolve address")

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
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
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
	TokenCreatedOn time.Time `json:"token_created_on"`
}

func (s Server) IP(ctx context.Context) (net.IP, error) {
	parsedIP := net.ParseIP(s.Address)
	if parsedIP != nil {
		// We already have an ip
		return parsedIP, nil
	}
	// TODO proper timeout for ctx
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.Address)
	if errResolve != nil || len(ips) == 0 {
		return nil, errors.Join(errResolve, ErrResolveIP)
	}

	return ips[0], nil
}

func (s Server) IPInternal(ctx context.Context) (net.IP, error) {
	parsedIP := net.ParseIP(s.AddressInternal)
	if parsedIP != nil {
		// We already have an ip
		return parsedIP, nil
	}
	// TODO proper timeout for ctx
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.Address)
	if errResolve != nil || len(ips) == 0 {
		return nil, errors.Join(errResolve, ErrResolveIP)
	}

	return ips[0], nil
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

type ServerQueryFilter struct {
	query.Filter
	IncludeDisabled bool `json:"include_disabled"`
	SDROnly         bool `json:"sdr_only"`
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

type PlayerServerInfo struct {
	Player   extra.Player
	ServerID int
}

type PartialStateUpdate struct {
	Hostname       string `json:"hostname"`
	ShortName      string `json:"short_name"`
	CurrentMap     string `json:"current_map"`
	PlayersReal    int    `json:"players_real"`
	PlayersTotal   int    `json:"players_total"`
	PlayersVisible int    `json:"players_visible"`
}

func NewServers(repository ServersRepository) Servers {
	return Servers{repository: repository}
}

type Servers struct {
	repository ServersRepository
}

// Delete performs a soft delete of the server. We use soft deleted because we dont wand to delete all the relationships
// that rely on this suchs a stats.
func (s *Servers) Delete(ctx context.Context, serverID int) error {
	if serverID <= 0 {
		return domain.ErrInvalidParameter
	}

	server, errServer := s.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	server.Deleted = true

	if err := s.repository.SaveServer(ctx, &server); err != nil {
		return err
	}

	slog.Info("Deleted server", slog.Int("server_id", serverID))

	return nil
}

func (s *Servers) Server(ctx context.Context, serverID int) (Server, error) {
	if serverID <= 0 {
		return Server{}, domain.ErrGetServer
	}

	return s.repository.GetServer(ctx, serverID)
}

func (s *Servers) ServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	return s.repository.GetServerPermissions(ctx)
}

func (s *Servers) Servers(ctx context.Context, filter ServerQueryFilter) ([]Server, int64, error) {
	return s.repository.GetServers(ctx, filter)
}

func (s *Servers) GetByName(ctx context.Context, serverName string, server *Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByName(ctx, serverName, server, disabledOk, deletedOk)
}

func (s *Servers) GetByPassword(ctx context.Context, serverPassword string, server *Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByPassword(ctx, serverPassword, server, disabledOk, deletedOk)
}

func (s *Servers) Save(ctx context.Context, req RequestServerUpdate) (Server, error) {
	var server Server

	if req.ServerID > 0 {
		existingServer, errServer := s.Server(ctx, req.ServerID)
		if errServer != nil {
			return Server{}, errServer
		}
		server = existingServer
		server.UpdatedOn = time.Now()
	} else {
		server = NewServer(req.ServerNameShort, req.Host, req.Port)
	}

	server.ShortName = req.ServerNameShort
	server.Name = req.ServerName
	server.Address = req.Host
	server.Port = req.Port
	server.ReservedSlots = req.ReservedSlots
	server.RCON = req.RCON
	server.Password = req.Password
	server.Latitude = req.Lat
	server.Longitude = req.Lon
	server.CC = req.CC
	server.Region = req.Region
	server.IsEnabled = req.IsEnabled
	server.LogSecret = req.LogSecret
	server.EnableStats = req.EnableStats
	server.AddressInternal = req.AddressInternal
	server.SDREnabled = req.SDREnabled

	if err := s.repository.SaveServer(ctx, &server); err != nil {
		return Server{}, err
	}

	if req.ServerID > 0 {
		slog.Info("Updated server successfully", slog.String("name", server.ShortName))
	} else {
		slog.Info("Created new server", slog.String("name", server.ShortName), slog.Int("server_id", server.ServerID))
	}

	return s.Server(ctx, server.ServerID)
}
