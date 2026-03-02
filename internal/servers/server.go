package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/rumblefrog/go-a2s"
	"golang.org/x/sync/errgroup"
)

var (
	maxPlayersRx = regexp.MustCompile(`^"sv_visiblemaxplayers" = "(\d{1,2})"\s`)
	playersRx    = regexp.MustCompile(`players\s: (\d+)\s+humans,\s+(\d+)\s+bots\s\((\d+)\s+max`)
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
	*sync.RWMutex

	// Auto generated id
	ServerID int `json:"server_id"`
	// ShortName is a short reference name for the server eg: us-1
	// This is used as a unique identifier for servers and is used for many different things such as paths,
	// so it's best to keep it short and without whitespace.
	ShortName string `json:"short_name"`
	// Name holds a default hostname. But its replaced with any updated title as they come in.
	Name string `json:"name"`
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
	// Password is what the sourcemod plugin on each server uses to generate a token to make authenticated calls.
	// This is *NOT* the general game server password (sv_password)
	Password  string `json:"password"`
	IsEnabled bool   `json:"is_enabled"`
	Deleted   bool   `json:"deleted"`
	Region    string `json:"region"`
	// CC is the 2 letter country code. Used for flags emojis.
	CC string `json:"cc"`
	// Physical Latitude location
	Latitude float64 `json:"latitude"`
	// Physical Longitude location
	Longitude float64 `json:"longitude"`
	// LogSecret is a unique integer used to "authenticate" UDP log packets.
	LogSecret   int  `json:"log_secret"`
	EnableStats bool `json:"enable_stats"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn time.Time `json:"token_created_on"`
	CreatedOn      time.Time `json:"created_on"`
	UpdatedOn      time.Time `json:"updated_on"`
	// DiscordSeedRoleIDs stores the discord role IDs for those who which to be notified of seed requests.
	DiscordSeedRoleIDs []string `json:"discord_seed_role_ids"` //nolint:tagliatelle
	// IP is distinct from Address as it can only contain a real IP and not DNS name like Address.
	IP net.IP `json:"ip"`
	// IPInternal works identical to IP except it uses the internal/VPN address from AddressInternal.
	// This is never exposed to client facing systems and is meant for when you want to communicate over
	// a VPN to the RCOn port instead of having it exposed publicly.
	IPInternal net.IP `json:"-"`

	lastMaxPlayersUpdate time.Time
	lastA2SUpdate        time.Time

	// state holds all information we know about the current dynamic game state of the server.
	state *state
}

func NewServer(shortName string, address string, port uint16) Server {
	return Server{
		RWMutex:            &sync.RWMutex{},
		state:              &state{Rules: map[string]string{}},
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

func (s *Server) resolveAll() error {
	waitGroup := errgroup.Group{}
	waitGroup.Go(s.resolveIP)
	waitGroup.Go(s.resolveIPInternal)

	if err := waitGroup.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrResolveIP, err)
	}

	return nil
}

func (s *Server) resolveIP() error {
	// Future: Make sure this is able to keep up with SDR changes.
	s.RLock()
	if s.IP != nil {
		s.RUnlock()

		return nil
	}

	ipAddr := net.ParseIP(s.Address)
	if ipAddr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), dnsResolveTimeout)
		defer cancel()

		ipAddr = resolveIP(ctx, s.Address)
	}

	if ipAddr == nil {
		s.RUnlock()

		return ErrResolveIP
	}
	s.RUnlock()

	s.Lock()
	s.IP = ipAddr
	s.Unlock()

	return nil
}

func (s *Server) resolveIPInternal() error {
	s.RLock()
	if s.AddressInternal == "" || s.IPInternal != nil {
		s.RUnlock()

		return nil
	}

	ipAddr := net.ParseIP(s.AddressInternal)
	if ipAddr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), dnsResolveTimeout)
		defer cancel()

		ipAddr = resolveIP(ctx, s.AddressInternal)
	}
	if ipAddr == nil {
		s.RUnlock()

		return ErrResolveIP
	}
	s.RUnlock()

	s.Lock()
	s.IPInternal = ipAddr
	s.Unlock()

	return nil
}

func (s *Server) LogAddressAdd(ctx context.Context, logAddress string) error {
	return s.ExecDiscardF(ctx, "logaddress_add %s", logAddress)
}

func (s *Server) LogAddressDel(ctx context.Context, logAddress string) error {
	return s.ExecDiscardF(ctx, "logaddress_add %s", logAddress)
}

type SayOpts struct {
	Type    SayType
	Message string
	Targets []steamid.SteamID
	Colour  SayColour
}

func (s *Server) Say(ctx context.Context, opts SayOpts) error {
	switch opts.Type {
	case Say:
		return s.ExecDiscardF(ctx, "sm_say %s", opts.Message)
	case CSay:
		return s.ExecDiscardF(ctx, "sm_csay %s", opts.Message)
	case TSay:
		return s.ExecDiscardF(ctx, `sm_tsay "%s" "%s"`, opts.Colour, opts.Message)
	case PSay:
		if len(opts.Targets) == 0 || len(opts.Targets) > 100 {
			return fmt.Errorf("%w: invalid steamid count for psay", ErrExecRCON)
		}
		for _, target := range opts.Targets {
			return s.ExecDiscardF(ctx, `sm_psay "#%s" "%s"`, target.Steam(false), opts.Message)
		}
	default:
		return fmt.Errorf("%w: invalid say type", ErrExecRCON)
	}

	return nil
}

func (s *Server) ExecDiscardF(ctx context.Context, command string, args ...any) error {
	return s.ExecDiscard(ctx, fmt.Sprintf(command, args...))
}

func (s *Server) ExecDiscard(ctx context.Context, command string) error {
	resp, err := s.Exec(ctx, command)
	if err != nil {
		return err
	}
	if resp != "" {
		slog.Debug("RCON Response", slog.String("command", command), slog.String("resp", resp))
	}

	return nil
}

func (s *Server) ExecF(ctx context.Context, command string, args ...any) (string, error) {
	return s.Exec(ctx, fmt.Sprintf(command, args...))
}

func (s *Server) Exec(ctx context.Context, command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("%w: empty command", ErrExecRCON)
	}

	s.RLock()
	addr := s.Addr()
	passwd := s.RCON
	s.RUnlock()

	conn, errConn := rcon.Dial(ctx, addr, passwd, serverQueryTimeout)
	if errConn != nil {
		return "", fmt.Errorf("%w: %w", ErrExecRCON, errConn)
	}
	defer log.Closer(conn)

	resp, errExec := conn.Exec(command)
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

	return s.ExecDiscardF(ctx, `sm_kick "#%s" %s`, target.Steam(false), reason)
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

	return s.ExecDiscardF(ctx, `sm_silence "#%s" %s`, target.Steam(false), reason)
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

func (s *Server) updateA2S() error {
	s.RLock()
	addr := s.Addr()
	s.RUnlock()

	client, errClient := a2s.NewClient(addr, a2s.TimeoutOption(serverQueryTimeout))
	if errClient != nil {
		return fmt.Errorf("%w: %w", ErrA2S, errClient)
	}
	defer func() {
		if errClose := client.Close(); errClose != nil {
			slog.Error("Failed to close a2s client", slog.String("error", errClose.Error()))
		}
	}()

	serverInfo, errQuery := client.QueryInfo()
	if errQuery != nil {
		return fmt.Errorf("%w: %w", ErrA2S, errQuery)
	}

	serverPlayer, errPlayer := client.QueryPlayer()
	if errPlayer != nil {
		return fmt.Errorf("%w: %w", ErrA2S, errPlayer)
	}

	serverRules, errRules := client.QueryRules()
	if errRules != nil {
		return fmt.Errorf("%w: %w", ErrA2S, errRules)
	}

	s.Lock()
	defer s.Unlock()

	s.state.Protocol = serverInfo.Protocol
	s.state.Folder = serverInfo.Folder
	s.state.Game = serverInfo.Game
	s.state.AppID = serverInfo.ID
	s.state.Version = serverInfo.Version
	s.state.VAC = serverInfo.VAC
	s.state.Bots = int(serverInfo.Bots)
	s.state.ServerOS = serverInfo.ServerOS.String()
	s.state.Password = !serverInfo.Visibility
	s.state.PlayerCount = int(serverInfo.Players)
	if serverInfo.SourceTV != nil {
		s.state.STVPort = serverInfo.SourceTV.Port
		s.state.STVName = serverInfo.SourceTV.Name
	}

	for _, player := range serverPlayer.Players {
		for _, p := range s.state.Players {
			if p.Name == player.Name {
				p.Score = int(player.Score)

				break
			}
		}
	}

	s.state.Rules = serverRules.Rules
	s.lastA2SUpdate = time.Now()

	return nil
}

func (s *Server) updateStatus(ctx context.Context) error {
	statusResp, errStatus := s.ExecF(ctx, "status")
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

	s.Lock()
	defer s.Unlock()

	// s.state.PlayerCount = status.PlayersCount
	// s.state.Bots = status.Bots
	players := make([]*Player, len(status.Players))
	for index, player := range status.Players {
		players[index] = &Player{
			UserID:        player.UserID,
			Name:          player.Name,
			SID:           player.SID,
			ConnectedTime: player.ConnectedTime,
			Ping:          player.Ping,
			Loss:          player.Loss,
			State:         player.State,
			IP:            player.IP,
			Port:          player.Port,
			Score:         0,
		}
	}
	s.state.Players = players
	if s.state.MaxPlayers == 0 && status.PlayersMax > 0 {
		// Prefer the sv_visiblemaxplayers value
		s.state.MaxPlayers = status.PlayersMax
	}
	s.state.Map = status.Map
	s.state.IP = status.IPInfo.FakeIP
	s.state.IPPublic = status.IPInfo.PublicIP
	s.state.Port = uint16(status.IPInfo.FakePort)         //nolint:gosec
	s.state.PortPublic = uint16(status.IPInfo.PublicPort) //nolint:gosec
	s.state.Tags = status.Tags
	s.state.Edicts = status.Edicts
	s.state.Version = status.Version
	s.state.STVIP = status.IPInfo.SourceTVIP
	s.state.STVPort = uint16(status.IPInfo.SourceTVFPort) //nolint:gosec
	if status.ServerName != "" {
		s.Name = status.ServerName
	}

	return nil
}

func (s *Server) updateMaxVisiblePlayers(ctx context.Context) error {
	maxPlayersResp, errMaxPlayers := s.ExecF(ctx, "sv_visiblemaxplayers")
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

	s.Lock()
	if maxPlayers > 0 {
		s.state.MaxPlayers = int(maxPlayers)
	}
	s.lastMaxPlayersUpdate = time.Now()
	s.Unlock()

	return nil
}

func (s *Server) updateState(ctx context.Context) {
	s.RLock()
	lastA2SUpdate := s.lastMaxPlayersUpdate
	lastPlayersUpdate := s.lastMaxPlayersUpdate
	s.RUnlock()

	waitGroup := &sync.WaitGroup{}

	waitGroup.Go(func() {
		if err := s.updateStatus(ctx); err != nil {
			slog.Error("Failed to parse status", slog.String("error", err.Error()),
				slog.String("server", s.ShortName))
		}
	})

	if time.Since(lastA2SUpdate) > time.Second*60 {
		waitGroup.Go(func() {
			if err := s.updateA2S(); err != nil {
				slog.Debug("Failed to update a2s", slog.String("error", err.Error()),
					slog.String("server", s.ShortName))

				return
			}
		})
	}

	if time.Since(lastPlayersUpdate) > time.Hour {
		waitGroup.Go(func() {
			if err := s.updateMaxVisiblePlayers(ctx); err != nil {
				slog.Warn("Got invalid max players value", slog.String("error", err.Error()),
					slog.String("server", s.ShortName))

				return
			}
		})
	}

	waitGroup.Wait()
}
