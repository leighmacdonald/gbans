package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrGetServer         = errors.New("failed to get server")
	ErrExecRCON          = errors.New("failed to execute rcon command")
	ErrResolveIP         = errors.New("failed to resolve address")
	ErrUpdateFreq        = errors.New("update freq must be >= 5s")
	ErrStatusParse       = errors.New("failed to parse status response")
	ErrMaxPlayerIntParse = errors.New("failed to cast max players value")
	ErrMaxPlayerParse    = errors.New("failed to parse sv_visiblemaxplayers response")
)

const (
	maxPlayersSupported     = 101
	DefaultStatusUpdateFreq = time.Second * 20
)

type SayType int

const (
	Say SayType = iota
	PSay
	CSay
	TSay
)

type SayColour string

const (
	White     SayColour = "white"
	Red       SayColour = "red"
	Green     SayColour = "green"
	Blue      SayColour = "blue"
	Yellow    SayColour = "yellow"
	Purple    SayColour = "purple"
	Cyan      SayColour = "cyan"
	Orange    SayColour = "orange"
	Pink      SayColour = "pink"
	Olive     SayColour = "olive"
	Lime      SayColour = "lime"
	Violet    SayColour = "violet"
	LightBlue SayColour = "lightblue"
)

type ServerFunc func(server *Server) error

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

type Servers struct {
	Repository

	logListener *logparse.Listener
	logFileChan chan LogFilePayload
	servers     Collection
	serversMu   *sync.RWMutex
	broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func New(repository Repository, broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent], logAddr string) (*Servers, error) {
	logSrc, errLogSrc := logparse.NewListener(logAddr,
		func(_ logparse.EventType, event logparse.ServerEvent) {
			broadcaster.Emit(event.EventType, event)
		})
	if errLogSrc != nil {
		return nil, errLogSrc
	}

	return &Servers{
		Repository:  repository,
		logListener: logSrc,
		logFileChan: make(chan LogFilePayload),
		servers:     Collection{},
		serversMu:   &sync.RWMutex{},
		broadcaster: broadcaster,
	}, nil
}

func (s *Servers) Current() []SafeServer {
	var curState []SafeServer //nolint:prealloc

	s.serversMu.RLock()
	defer s.serversMu.RUnlock()

	for _, srv := range s.servers {
		if !srv.Deleted && srv.IsEnabled {
			srv.RLock()
			curState = append(curState, SafeServer{
				Host: srv.Address,
				Port: srv.Port,
				// TODO
				// IP:         srv.IP(),
				Name:       srv.Name,
				NameShort:  srv.ShortName,
				Region:     srv.Region,
				CC:         srv.CC,
				ServerID:   srv.ServerID,
				Players:    srv.state.PlayerCount,
				MaxPlayers: srv.state.MaxPlayers,
				Bots:       srv.state.Bots,
				Humans:     srv.state.Humans,
				Map:        srv.state.Map,
				Tags:       srv.state.Tags,
				GameTypes:  []string{},
				Latitude:   srv.Latitude,
				Longitude:  srv.Longitude,
			})
			srv.RUnlock()
		}
	}

	sort.SliceStable(curState, func(i, j int) bool {
		return curState[i].NameShort < curState[j].NameShort
	})

	return curState
}

func (s *Servers) Broadcast(ctx context.Context, cmd string, args ...any) {
	if cmd == "" {
		return
	}
	s.servers.broadcast(ctx, fmt.Sprintf(cmd, args...))
}

func (s *Servers) FindPlayers(opts FindOpts) []FindResult {
	return s.servers.find(opts)
}

func (s *Servers) FindPlayer(opts FindOpts) (FindResult, bool) {
	results := s.servers.find(opts)
	if len(results) == 0 {
		return FindResult{}, false
	}

	return results[0], true
}

func (s *Servers) Start(ctx context.Context, updateFreq time.Duration) error {
	if updateFreq < time.Second*5 {
		return ErrUpdateFreq
	}

	var (
		ticker  = time.NewTicker(updateFreq)
		timeOut = time.Duration(float64(updateFreq) * 0.8)
	)

	go s.logListener.Start(ctx)

	for {
		select {
		case <-ticker.C:
			timeout, cancel := context.WithTimeout(ctx, timeOut)
			if err := s.updateStates(timeout); err != nil {
				slog.Error("Failed to update server states", slog.String("error", err.Error()))
			}
			cancel()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Servers) updateStates(ctx context.Context) error {
	var (
		waitGroup  = &sync.WaitGroup{}
		successful = atomic.Int32{}
		existing   = atomic.Int32{}
		startTIme  = time.Now()
	)

	servers, errServers := s.Servers(ctx, Query{})
	if errServers != nil {
		return errServers
	}

	var valid []*Server
	s.serversMu.Lock()
	for _, server := range servers {
		found := false
		for _, existingServer := range s.servers {
			if existingServer.ServerID == server.ServerID {
				valid = append(valid, existingServer)
				found = true

				break
			}
		}
		if !found {
			valid = append(valid, &server)
		}
	}
	s.servers = valid
	s.serversMu.Unlock()

	for _, server := range s.servers {
		waitGroup.Go(func() {
			server.updateState(ctx)
			successful.Add(1)
		})
	}

	waitGroup.Wait()

	if fail := len(s.servers) - int(successful.Load()); fail > 0 {
		slog.Debug("RCON update cycle complete",
			slog.Int("success", int(successful.Load())),
			slog.Int("existing", int(existing.Load())),
			slog.Int("fail", fail),
			slog.Duration("duration", time.Since(startTIme)))
	}

	return nil
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

	if err := s.Repository.Save(ctx, &server); err != nil {
		return err
	}

	return nil
}

func (s *Servers) Server(ctx context.Context, serverID int) (Server, error) {
	if serverID <= 0 {
		return Server{}, ErrNotFound
	}

	servers, err := s.Query(ctx, Query{ServerID: serverID, IncludeDisabled: true})
	if err != nil {
		return Server{}, err
	}

	if len(servers) != 1 {
		return Server{}, ErrNotFound
	}

	return servers[0], nil
}

func (s *Servers) ServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	return s.GetServerPermissions(ctx)
}

func (s *Servers) Servers(ctx context.Context, filter Query) ([]Server, error) {
	return s.Query(ctx, filter)
}

func (s *Servers) GetByName(ctx context.Context, serverName string) (Server, error) {
	server, errServer := s.Query(ctx, Query{ShortName: serverName})

	if errServer != nil {
		return Server{}, errServer
	}

	if len(server) == 0 {
		return Server{}, ErrUnknownServer
	}

	return server[0], nil
}

func (s *Servers) GetByPassword(ctx context.Context, serverPassword string) (Server, error) {
	server, errServer := s.Query(ctx, Query{Password: serverPassword})

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

	if err := s.Repository.Save(ctx, &server); err != nil {
		return Server{}, err
	}

	return s.Server(ctx, server.ServerID)
}

func (s *Servers) AutoCompleteServers(ctx context.Context, query string) ([]discord.AutoCompleteValuer, error) {
	activeServers, errServer := s.Servers(ctx, Query{})
	if errServer != nil {
		return nil, errServer
	}
	query = strings.ToLower(query)
	var values []discord.AutoCompleteValuer //nolint:prealloc
	for _, server := range activeServers {
		if query == "" ||
			query == "*" ||
			strings.Contains(strings.ToLower(server.Name), query) ||
			strings.Contains(strings.ToLower(server.ShortName), query) {
			values = append(values, discord.NewAutoCompleteValue(server.Name, server.ShortName))
		}
	}

	return values, nil
}

func (s *Servers) Each(serverFn ServerFunc) {
	s.serversMu.RLock()
	defer s.serversMu.RUnlock()

	waitGroup := &sync.WaitGroup{}
	for _, server := range s.servers {
		waitGroup.Go(func() {
			if err := serverFn(server); err != nil {
				slog.Error("Failed to execute server fn",
					slog.String("server", server.ShortName),
					slog.String("error", err.Error()))
			}
		})
	}
	waitGroup.Wait()
}

func ShortNamePrefix(name string) string {
	pieces := strings.Split(name, "-")

	return strings.Join(pieces[0:len(pieces)-1], "-")
}

func resolveIP(ctx context.Context, addr string) net.IP {
	if ipAddr := net.ParseIP(addr); ipAddr != nil {
		return ipAddr
	}

	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", addr)
	if errResolve != nil || len(ips) == 0 {
		return nil
	}

	return ips[0]
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radianLat1 := math.Pi * lat1 / 180
	radianLat2 := math.Pi * lat2 / 180
	theta := lng1 - lng2
	radianTheta := math.Pi * theta / 180

	dist := math.Sin(radianLat1)*math.Sin(radianLat2) + math.Cos(radianLat1)*math.Cos(radianLat2)*math.Cos(radianTheta)
	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515
	dist *= 1.609344 // convert to km

	return dist
}
