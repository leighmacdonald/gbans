package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/ryanuber/go-glob"
	"golang.org/x/sync/errgroup"
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

type State struct {
	state       *StateRepository
	config      *config.Configuration
	servers     Servers
	logListener *logparse.UDPLogListener
	logFileChan chan LogFilePayload
	broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

// NewState created a interface to interact with server state and exec rcon commands.
func NewState(broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository *StateRepository, config *config.Configuration, servers Servers,
) *State {
	return &State{
		state:       repository,
		config:      config,
		broadcaster: broadcaster,
		servers:     servers,
		logFileChan: make(chan LogFilePayload),
	}
}

func (s *State) Start(ctx context.Context) error {
	conf := s.config.Config()

	logSrc, errLogSrc := logparse.NewUDPLogListener(conf.General.SrcdsLogAddr,
		func(_ logparse.EventType, event logparse.ServerEvent) {
			s.broadcaster.Emit(event.EventType, event)
		})

	if errLogSrc != nil {
		return errLogSrc
	}

	s.logListener = logSrc

	go s.state.Start(ctx)

	// TODO run on server Config changes
	s.updateSrcdsLogServers(ctx)

	s.logListener.Start(ctx)

	return nil
}

func (s *State) updateSrcdsLogServers(ctx context.Context) {
	newSecrets := map[int]logparse.ServerIDMap{}
	newServers := map[netip.Addr]bool{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, _, errServers := s.servers.Servers(serversCtx, ServerQueryFilter{
		IncludeDisabled: false,
		Filter:          query.Filter{Deleted: false},
	})
	if errServers != nil {
		slog.Error("Failed to update srcds log secrets", log.ErrAttr(errServers))

		return
	}

	for _, server := range servers {
		newSecrets[server.LogSecret] = logparse.ServerIDMap{
			ServerID:   server.ServerID,
			ServerName: server.ShortName,
		}

		if ip, errIP := server.IP(ctx); errIP == nil {
			if addr, addrOk := netip.AddrFromSlice(ip); addrOk {
				newServers[addr] = true
			} else {
				slog.Error("Failed to convert ip", slog.String("address", ip.String()))
			}
		} else {
			slog.Error("Failed to resolve server ip", log.ErrAttr(errIP))
		}

		if internalIP, errIP := server.IPInternal(ctx); errIP == nil {
			if addr, addrOk := netip.AddrFromSlice(internalIP); addrOk {
				newServers[addr] = true
			} else {
				slog.Error("Failed to convert internal ip", slog.String("address", internalIP.String()))
			}
		} else {
			slog.Error("Failed to resolve internal server ip", log.ErrAttr(errIP))
		}
	}

	s.logListener.SetSecrets(newSecrets)
	s.logListener.SetServers(newServers)
}

func (s *State) Current() []ServerState {
	return s.state.Current()
}

func (s *State) FindByCIDR(cidr *net.IPNet) []PlayerServerInfo {
	return s.Find("", steamid.SteamID{}, nil, cidr)
}

func (s *State) FindByIP(addr net.IP) []PlayerServerInfo {
	return s.Find("", steamid.SteamID{}, addr, nil)
}

func (s *State) FindByName(name string) []PlayerServerInfo {
	return s.Find(name, steamid.SteamID{}, nil, nil)
}

func (s *State) FindBySteamID(steamID steamid.SteamID) []PlayerServerInfo {
	return s.Find("", steamID, nil, nil)
}

func (s *State) Update(serverID int, update PartialStateUpdate) error {
	return s.state.Update(serverID, update)
}

// Find searches the current server state for players matching at least one of the provided criteria.
func (s *State) Find(name string, steamID steamid.SteamID, addr net.IP, cidr *net.IPNet) []PlayerServerInfo {
	var found []PlayerServerInfo

	current := s.state.Current()

	for server := range current {
		for _, player := range current[server].Players {
			matched := false
			if steamID.Valid() && player.SID == steamID {
				matched = true
			}

			if name != "" {
				queryName := name
				if !strings.HasPrefix(queryName, "*") {
					queryName = "*" + queryName
				}

				if !strings.HasSuffix(queryName, "*") {
					queryName += "*"
				}

				m := glob.Glob(strings.ToLower(queryName), strings.ToLower(player.Name))
				if m {
					matched = true
				}
			}

			if addr != nil && addr.Equal(player.IP) {
				matched = true
			}

			if cidr != nil && cidr.Contains(player.IP) {
				matched = true
			}

			if matched {
				found = append(found, PlayerServerInfo{Player: player, ServerID: current[server].ServerID})
			}
		}
	}

	return found
}

func (s *State) SortRegion() map[string][]ServerState {
	serverMap := map[string][]ServerState{}
	for _, server := range s.state.Current() {
		_, exists := serverMap[server.Region]
		if !exists {
			serverMap[server.Region] = []ServerState{}
		}

		serverMap[server.Region] = append(serverMap[server.Region], server)
	}

	return serverMap
}

func (s *State) ByServerID(serverID int) (ServerState, bool) {
	for _, server := range s.state.Current() {
		if server.ServerID == serverID {
			return server, true
		}
	}

	return ServerState{}, false
}

func (s *State) ByName(name string, wildcardOk bool) []ServerState {
	var servers []ServerState

	current := s.state.Current()

	if name == "*" && wildcardOk {
		servers = append(servers, current...)
	} else {
		if !strings.HasPrefix(name, "*") {
			name = "*" + name
		}

		if !strings.HasSuffix(name, "*") {
			name += "*"
		}

		for _, server := range current {
			if glob.Glob(strings.ToLower(name), strings.ToLower(server.NameShort)) ||
				strings.EqualFold(server.NameShort, name) {
				servers = append(servers, server)

				break
			}
		}
	}

	return servers
}

func (s *State) ServerIDsByName(name string, wildcardOk bool) []int {
	var servers []int //nolint:prealloc
	for _, server := range s.ByName(name, wildcardOk) {
		servers = append(servers, server.ServerID)
	}

	return servers
}

func (s *State) OnFindExec(ctx context.Context, name string, steamID steamid.SteamID, ip net.IP, cidr *net.IPNet, onFoundCmd func(info PlayerServerInfo) string) error {
	currentState := s.state.Current()
	players := s.Find(name, steamID, ip, cidr)

	if len(players) == 0 {
		return domain.ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		for _, server := range currentState {
			if player.ServerID == server.ServerID {
				_, errRcon := s.ExecServer(ctx, server.ServerID, onFoundCmd(player))
				if errRcon != nil {
					err = errors.Join(errRcon)
				}
			}
		}
	}

	return err
}

func (s *State) ExecServer(ctx context.Context, serverID int, cmd string) (string, error) {
	var conf ServerConfig

	for _, server := range s.state.Configs() {
		if server.ServerID == serverID {
			conf = server

			break
		}
	}

	if conf.ServerID == 0 {
		return "", ErrUnknownServerID
	}

	return s.ExecRaw(ctx, conf.Addr(), conf.RconPassword, cmd)
}

func (s *State) ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error) {
	return s.state.ExecRaw(ctx, addr, password, cmd)
}

func (s *State) LogAddressAdd(ctx context.Context, logAddress string) {
	time.Sleep(20 * time.Second)
	slog.Info("Enabling log forwarding", slog.String("host", logAddress))
	s.Broadcast(ctx, nil, "logaddress_add "+logAddress)
}

func (s *State) LogAddressDel(ctx context.Context, logAddress string) {
	slog.Info("Disabling log forwarding for host", slog.String("host", logAddress))
	s.Broadcast(ctx, nil, "logaddress_add "+logAddress)
}

type broadcastResult struct {
	serverID int
	resp     string
}

// Broadcast sends out rcon commands to all provided servers. If no servers are provided it will default to broadcasting
// to every server.
func (s *State) Broadcast(ctx context.Context, serverIDs []int, cmd string) map[int]string {
	results := map[int]string{}
	errGroup, egCtx := errgroup.WithContext(ctx)

	configs := s.state.Configs()

	if len(serverIDs) == 0 {
		for _, conf := range configs {
			serverIDs = append(serverIDs, conf.ServerID)
		}
	}

	resultChan := make(chan broadcastResult)

	for _, serverID := range serverIDs {
		sid := serverID

		errGroup.Go(func() error {
			serverConf, errServerConf := s.state.GetServer(sid)
			if errServerConf != nil {
				return errServerConf
			}

			resp, errExec := s.state.ExecRaw(egCtx, serverConf.Addr(), serverConf.RconPassword, cmd)
			if errExec != nil {
				if errors.Is(errExec, context.Canceled) {
					return nil
				}

				slog.Error("Failed to exec server command", slog.String("name", serverConf.DefaultHostname),
					slog.Int("server_id", sid), log.ErrAttr(errExec))

				// Don't error out since we don't want a single servers potentially temporary issue to prevent the rest
				// from executing.
				return nil
			}

			resultChan <- broadcastResult{
				serverID: sid,
				resp:     resp,
			}

			return nil
		})
	}

	go func() {
		err := errGroup.Wait()
		if err != nil {
			slog.Error("Failed to broadcast command", log.ErrAttr(err))
		}

		close(resultChan)
	}()

	for result := range resultChan {
		results[result.serverID] = result.resp
	}

	return results
}

// Kick will kick the steam id from whatever server it is connected to.
func (s *State) Kick(ctx context.Context, target steamid.SteamID, reason string) error {
	if !target.Valid() {
		return domain.ErrInvalidTargetSID
	}

	if errExec := s.OnFindExec(ctx, "", target, nil, nil, func(info PlayerServerInfo) string {
		return fmt.Sprintf("sm_kick #%d %s", info.Player.UserID, reason)
	}); errExec != nil {
		return errExec
	}

	return nil
}

// KickPlayerID will kick the steam id from whatever server it is connected to.
func (s *State) KickPlayerID(ctx context.Context, targetPlayerID int, targetServerID int, reason string) error {
	_, err := s.ExecServer(ctx, targetServerID, fmt.Sprintf("sm_kick #%d %s", targetPlayerID, reason))

	return err
}

// Silence will gag & mute a player.
func (s *State) Silence(ctx context.Context, target steamid.SteamID, reason string,
) error {
	if !target.Valid() {
		return domain.ErrInvalidTargetSID
	}

	var (
		users   []string
		usersMu = &sync.RWMutex{}
	)

	if errExec := s.OnFindExec(ctx, "", target, nil, nil, func(info PlayerServerInfo) string {
		usersMu.Lock()
		users = append(users, info.Player.Name)
		usersMu.Unlock()

		return fmt.Sprintf(`sm_silence "#%s" %s`, info.Player.SID.Steam(false), reason)
	}); errExec != nil {
		return errors.Join(errExec, fmt.Errorf("%w: sm_silence", errExec))
	}

	return nil
}

// Say is used to send a message to the server via sm_say.
func (s *State) Say(ctx context.Context, serverID int, message string) error {
	_, errExec := s.ExecServer(ctx, serverID, `sm_say `+message)

	return errors.Join(errExec, fmt.Errorf("%w: sm_say", errExec))
}

// CSay is used to send a centered message to the server via sm_csay.
func (s *State) CSay(ctx context.Context, serverID int, message string) error {
	_, errExec := s.ExecServer(ctx, serverID, `sm_csay `+message)

	return errors.Join(errExec, fmt.Errorf("%w: sm_csay", errExec))
}

// PSay is used to send a private message to a player.
func (s *State) PSay(ctx context.Context, target steamid.SteamID, message string) error {
	if !target.Valid() {
		return domain.ErrInvalidTargetSID
	}

	if errExec := s.OnFindExec(ctx, "", target, nil, nil, func(_ PlayerServerInfo) string {
		return fmt.Sprintf(`sm_psay "#%s" "%s"`, target.Steam(false), message)
	}); errExec != nil {
		return errors.Join(errExec, fmt.Errorf("%w: sm_psay", errExec))
	}

	return nil
}
