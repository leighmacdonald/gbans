package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	maxPlayersRx = regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`)
	playersRx    = regexp.MustCompile(`players\s: (\d+)\s+humans,\s+(\d+)\s+bots\s\((\d+)\s+max`)
)

type SayType int

const (
	Say SayType = iota
	PSay
	CSay
)

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
	Tags       []string `json:"tags"`
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

	resolvedIP           net.IP
	lastMaxPlayersUpdate time.Time
	state                *State
}

func NewServer(shortName string, address string, port uint16) Server {
	return Server{
		state:              &State{},
		ShortName:          shortName,
		Address:            address,
		Port:               port,
		RCON:               stringutil.SecureRandomString(10),
		ReservedSlots:      0,
		Password:           stringutil.SecureRandomString(10),
		IsEnabled:          true,
		EnableStats:        true,
		TokenCreatedOn:     time.Unix(0, 0),
		CreatedOn:          time.Now(),
		UpdatedOn:          time.Now(),
		DiscordSeedRoleIDs: []string{},
	}
}

func (s *Server) IP(ctx context.Context) (string, error) {
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.Address)
	if errResolve != nil || len(ips) == 0 {
		return "", errors.Join(errResolve, ErrResolveIP)
	}

	return ResolveIP(ctx, s.Address)
}

func (s *Server) IPInternal(ctx context.Context) (string, error) {
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.AddressInternal)
	if errResolve != nil || len(ips) == 0 {
		return "", errors.Join(errResolve, ErrResolveIP)
	}

	return ResolveIP(ctx, s.Address)
}

func (s *Server) LogAddressAdd(ctx context.Context, logAddress string) error {
	return s.ExecDiscard(ctx, "logaddress_add %s", logAddress)
}

func (s *Server) LogAddressDel(ctx context.Context, logAddress string) error {
	return s.ExecDiscard(ctx, "logaddress_add %s", logAddress)
}

func (s *Server) Say(ctx context.Context, sayType SayType, message string, steamIDs ...steamid.SteamID) error {
	var cmd string
	switch sayType {
	case Say:
		cmd = "sm_say"
	case CSay:
		cmd = "sm_csay"
	case PSay:
		if len(steamIDs) == 0 || len(steamIDs) > 100 {
			return fmt.Errorf("%w: invalid steamid count for psay", ErrExecRCON)
		}

		return s.ExecDiscard(ctx, `sm_psay "#%s" "%s"`, cmd, message)
	default:
		return fmt.Errorf("%w: invalid say type", ErrExecRCON)
	}

	return s.ExecDiscard(ctx, "%s %s", cmd, message)
}

func (s *Server) ExecDiscard(ctx context.Context, command string, args ...any) error {
	resp, err := s.Exec(ctx, command, args...)
	if err != nil {
		return err
	}
	if resp != "" {
		slog.Debug("RCON Response", slog.String("command", command), slog.String("resp", resp))
	}

	return nil
}

func (s *Server) Exec(ctx context.Context, command string, args ...any) (string, error) {
	if command == "" {
		return "", fmt.Errorf("%w: empty command", ErrExecRCON)
	}

	conn, errConn := rcon.Dial(ctx, s.Addr(), s.RCON, time.Second*5)
	if errConn != nil {
		return "", fmt.Errorf("%w: %w", ErrExecRCON, errConn)
	}
	defer log.Closer(conn)

	resp, errExec := conn.Exec(fmt.Sprintf(command, args))
	if errExec != nil {
		return "", fmt.Errorf("%w: %w", ErrExecRCON, errExec)
	}

	return resp, nil
}

// Kick will kick the steam id from whatever server it is connected to.
func (s *Server) Kick(ctx context.Context, target steamid.SteamID, reason string) error {
	if !target.Valid() {
		return steamid.ErrInvalidSID
	}

	return s.ExecDiscard(ctx, `sm_kick "#%s" %s`, target.Steam(false), reason)
}

// KickPlayerID will kick the steam id from whatever server it is connected to.
func (s *Server) KickPlayerID(ctx context.Context, targetPlayerID int, reason string) error {
	return s.ExecDiscard(ctx, fmt.Sprintf("sm_kick #%d %s", targetPlayerID, reason))
}

// Silence will gag & mute a player.
func (s *Server) Silence(ctx context.Context, target steamid.SteamID, reason string) error {
	if !target.Valid() {
		return steamid.ErrInvalidSID
	}

	return s.ExecDiscard(ctx, `sm_silence "#%s" %s`, target.Steam(false), reason)
}

func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s *Server) AddrInternalOrDefault() string {
	if s.AddressInternal != "" {
		return s.AddressInternal
	}

	return s.Address
}

func (s *Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

func (s *Server) Connect() string {
	return link.Raw(fmt.Sprintf("/connect/%d", s.ServerID))
}

func (s *Server) SteamLink() string {
	ipAddr, err := net.ResolveIPAddr("ip4", s.Address)
	if err != nil {
		slog.Error("Failed to resolve ip4", slog.String("error", err.Error()))

		return fmt.Sprintf("steam://run/440//+connect %s:%d", s.Address, s.Port)
	}

	return fmt.Sprintf("steam://run/440//+connect %s:%d", ipAddr.String(), s.Port)
}

func (s *Server) updateStatus(ctx context.Context) error {
	statusResp, errStatus := s.Exec(ctx, "status")
	if errStatus != nil {
		return errStatus
	}

	pStatus, errParse := extra.ParseStatus(statusResp, true)
	if errParse != nil {
		return errors.Join(errParse, ErrStatusParse)
	}

	status := Status{Status: pStatus}
	matches := playersRx.FindStringSubmatch(statusResp)
	if len(matches) > 0 {
		players, errPlayers := strconv.Atoi(matches[1])
		if errPlayers != nil {
			return ErrStatusParse
		}
		status.Humans = players

		bots, errBots := strconv.Atoi(matches[2])
		if errBots != nil {
			return ErrStatusParse
		}
		status.Bots = bots

		maxPlayers, errMaxPlayers := strconv.Atoi(matches[3])
		if errMaxPlayers != nil {
			return ErrStatusParse
		}

		if maxPlayers%2 != 0 {
			// Assume that if we have an uneven player count that it's a SourceTV instance and ignore it.
			status.Bots--
			maxPlayers--
		}

		status.PlayersMax = maxPlayers
	}

	s.state.PlayerCount = status.PlayersCount
	s.state.Bots = status.Bots
	s.state.Players = status.Players
	s.state.MaxPlayers = status.PlayersMax
	s.state.Map = status.Map
	s.state.IP = status.IPInfo.FakeIP
	s.state.IPPublic = status.IPInfo.PublicIP
	s.state.Port = uint16(status.IPInfo.FakePort)
	s.state.PortPublic = uint16(status.IPInfo.PublicPort)
	s.state.Tags = status.Tags
	s.state.Edicts = status.Edicts
	s.state.Version = status.Version
	s.state.STVIP = status.IPInfo.SourceTVIP
	s.state.STVPort = uint16(status.IPInfo.SourceTVFPort)
	if status.ServerName != "" {
		s.Name = status.ServerName
	}

	return nil
}

func (s *Server) updateMaxVisiblePlayers(ctx context.Context) error {
	maxPlayersResp, errMaxPlayers := s.Exec(ctx, "sv_visiblemaxplayers")
	if errMaxPlayers != nil {
		return errMaxPlayers
	}

	matches := maxPlayersRx.FindStringSubmatch(maxPlayersResp)
	if matches == nil || len(matches) != 2 {
		return ErrMaxPlayerParse
	}

	maxPlayers, errCast := strconv.ParseInt(matches[1], 10, 32)
	if errCast != nil {
		return errors.Join(errCast, ErrMaxPlayerIntParse)
	}

	if maxPlayers > maxPlayersSupported {
		maxPlayers = -1
	}

	s.state.MaxPlayers = int(maxPlayers)
	s.lastMaxPlayersUpdate = time.Now()

	return nil
}
