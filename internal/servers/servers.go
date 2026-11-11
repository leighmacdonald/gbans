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

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

var (
	ErrGetServer         = errors.New("failed to get server")
	ErrExecRCON          = errors.New("failed to execute rcon command")
	ErrResolveIP         = errors.New("failed to resolve address")
	ErrUpdateFreq        = errors.New("update freq must be >= 5s")
	ErrStatusParse       = errors.New("failed to parse status response")
	ErrMaxPlayerIntParse = errors.New("failed to cast max players value")
	ErrMaxPlayerParse    = errors.New("failed to parse sv_visiblemaxplayers response")
	ErrNotFound          = errors.New("server not found")
	ErrServerDisabled    = errors.New("server is disabled")
	ErrInvalidSeedRole   = errors.New("invalid seed role")
)

const (
	maxPlayersSupported     = 101
	DefaultStatusUpdateFreq = time.Second * 20
	serverQueryTimeout      = time.Second * 5
	dnsResolveTimeout       = time.Second * 5

	// logSecretCacheTTL bounds how long a server's log secret and resolved
	// address are trusted without re-reading them from the database.
	logSecretCacheTTL = time.Minute
	// logSecretMissTTL limits repeated database lookups for unknown secrets.
	logSecretMissTTL = 10 * time.Second
	// maxSecretCacheEntries caps the cache against floods of random secrets.
	maxSecretCacheEntries = 4096
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
	ServerID        int32  `query:"server_id"`
	IncludeDisabled bool   `query:"include_disabled"`
	SDROnly         bool   `query:"sdr_only"`
	ShortName       string `query:"short_name"`
	Password        string `query:"password"`
	IncludeDeleted  bool
}

type ServerInfoSafe struct {
	ServerNameLong string
	ServerName     string
	ServerID       int32
	Colour         string
}

type Servers struct {
	repo Repository

	logListener *logparse.Listener
	logFileChan chan LogFilePayload
	servers     Collection
	serversMu   *sync.RWMutex
	broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
	logAddr     string
	logRecorder *LogEventRecorder
	logBuffer   *LogBuffer

	secretMu    *sync.RWMutex
	secretCache map[int64]secretAuthEntry
}

// secretAuthEntry caches the result of authenticating a srcds log secret. A
// zero serverID marks a negative entry so floods of unknown secrets do not
// query the database for every packet.
type secretAuthEntry struct {
	serverID   int32
	serverName string
	ip         net.IP
	expiresAt  time.Time
}

func New(repository Repository, broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent], logAddr string) (*Servers, error) {
	servers := &Servers{
		repo:        repository,
		logFileChan: make(chan LogFilePayload),
		servers:     Collection{},
		serversMu:   &sync.RWMutex{},
		broadcaster: broadcaster,
		logAddr:     logAddr,
		logRecorder: newLogEventRecorder(repository),
		logBuffer:   NewLogBuffer(),
		secretMu:    &sync.RWMutex{},
		secretCache: make(map[int64]secretAuthEntry),
	}

	return servers, nil
}

func (s *Servers) onLogEvent(_ logparse.EventType, event logparse.ServerEvent) {
	s.logBuffer.Push(event.ServerID, event)
	s.broadcaster.Emit(event.EventType, event)
	s.logRecorder.send(event)
}

// LogBuffer exposes the in-memory per-server log line buffer.
func (s *Servers) LogBuffer() *LogBuffer {
	return s.logBuffer
}

func (s *Servers) QueryLogs(ctx context.Context, opts QueryLogOpts) ([]ServerLog, int64, error) {
	return s.repo.QueryLogs(ctx, opts)
}

func (s *Servers) secretAuth(ctx context.Context, secret int64, ipAddr net.IP) (int32, string, error) {
	entry, cached := s.cachedSecretAuth(secret)
	if !cached {
		server, err := s.repo.ServerByLogSecret(ctx, secret)
		if err != nil {
			s.storeSecretAuth(secret, secretAuthEntry{expiresAt: time.Now().Add(logSecretMissTTL)})

			return 0, "", fmt.Errorf("%w: invalid log_secret", err)
		}

		entry = secretAuthEntry{
			serverID:   server.ServerID,
			serverName: server.ShortName,
			ip:         resolveServerIP(ctx, server),
			expiresAt:  time.Now().Add(logSecretCacheTTL),
		}

		s.storeSecretAuth(secret, entry)
	}

	if entry.serverID == 0 {
		return 0, "", fmt.Errorf("%w: invalid log_secret", ErrNotFound)
	}

	if entry.ip == nil || !entry.ip.Equal(ipAddr) {
		return 0, "", fmt.Errorf("%w: invalid source ip", ErrNotFound)
	}

	return entry.serverID, entry.serverName, nil
}

func (s *Servers) cachedSecretAuth(secret int64) (secretAuthEntry, bool) {
	s.secretMu.RLock()
	entry, ok := s.secretCache[secret]
	s.secretMu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return secretAuthEntry{}, false
	}

	return entry, true
}

func (s *Servers) storeSecretAuth(secret int64, entry secretAuthEntry) {
	s.secretMu.Lock()
	defer s.secretMu.Unlock()

	if len(s.secretCache) >= maxSecretCacheEntries {
		now := time.Now()
		for key, cached := range s.secretCache {
			if now.After(cached.expiresAt) {
				delete(s.secretCache, key)
			}
		}

		if len(s.secretCache) >= maxSecretCacheEntries {
			clear(s.secretCache)
		}
	}

	s.secretCache[secret] = entry
}

func (s *Servers) invalidateSecretCache() {
	s.secretMu.Lock()
	defer s.secretMu.Unlock()

	clear(s.secretCache)
}

// resolveServerIP returns the address log packets are expected to originate
// from. The internal/VPN address takes precedence when configured. Resolution
// happens once per cache entry instead of once per log packet.
func resolveServerIP(ctx context.Context, server Server) net.IP {
	addr := server.Address
	if server.AddressInternal != "" {
		addr = server.AddressInternal
	}

	resolveCtx, cancel := context.WithTimeout(ctx, dnsResolveTimeout)
	defer cancel()

	return resolveIP(resolveCtx, addr)
}

func (s *Servers) Current() []SafeServer {
	var curState []SafeServer //nolint:prealloc

	s.serversMu.RLock()
	defer s.serversMu.RUnlock()

	for _, srv := range s.servers {
		if !srv.Deleted && srv.IsEnabled {
			srv.RLock()
			curState = append(curState, srv.snapshot())
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
	s.serversMu.RLock()
	defer s.serversMu.RUnlock()

	return s.servers.find(opts)
}

func (s *Servers) FindPlayer(opts FindOpts) (FindResult, bool) {
	results := s.FindPlayers(opts)
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

	logSrc, errLogSrc := logparse.NewListener(s.logAddr, s.onLogEvent, s.secretAuth)
	if errLogSrc != nil {
		return errLogSrc
	}

	go logSrc.Start(ctx)

	go s.logRecorder.start(ctx)

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
			if err := server.resolveAll(); err != nil {
				slog.Error("Failed to resolve server IPs",
					slog.String("error", err.Error()), slog.String("server", server.ShortName))

				continue
			}
			valid = append(valid, &server)
		}
	}
	s.servers = valid
	s.serversMu.Unlock()

	for _, server := range valid {
		waitGroup.Go(func() {
			server.updateState(ctx)
			successful.Add(1)
		})
	}

	waitGroup.Wait()

	if fail := len(valid) - int(successful.Load()); fail > 0 {
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
func (s *Servers) Delete(ctx context.Context, serverID int32) error {
	if serverID <= 0 {
		return httphelper.ErrInvalidParameter
	}

	server, errServer := s.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	server.Deleted = true

	if err := s.repo.Save(ctx, &server); err != nil {
		return err
	}

	s.invalidateSecretCache()

	return nil
}

func (s *Servers) Server(ctx context.Context, serverID int32) (Server, error) {
	if serverID <= 0 {
		return Server{}, ErrNotFound
	}

	servers, err := s.repo.Query(ctx, Query{ServerID: serverID, IncludeDisabled: true})
	if err != nil {
		return Server{}, err
	}

	if len(servers) != 1 {
		return Server{}, ErrNotFound
	}

	return servers[0], nil
}

// SafeState returns the live state of the given server if it is currently being polled, or a
// static state derived from the database row otherwise. This lets callers that only need basic
// server info (e.g. the seed request) work before the status poll loop has resolved the server.
// Servers that are not enabled (or not found) return ErrNotFound.
func (s *Servers) SafeState(ctx context.Context, serverID int32) (SafeServer, error) {
	s.serversMu.RLock()
	live, ok := s.servers.byServerID(serverID)
	s.serversMu.RUnlock()
	if ok {
		live.RLock()
		snapshot := live.snapshot()
		live.RUnlock()

		return snapshot, nil
	}

	server, errServer := s.Server(ctx, serverID)
	if errServer != nil {
		return SafeServer{}, errServer
	}

	if !server.IsEnabled {
		return SafeServer{}, ErrNotFound
	}

	ipAddr := ""
	if server.IP != nil {
		ipAddr = server.IP.String()
	}

	return SafeServer{
		ServerID:  server.ServerID,
		Host:      server.Address,
		Port:      server.Port,
		Name:      server.Name,
		NameShort: server.ShortName,
		IP:        ipAddr,
		CC:        server.CC,
		Region:    server.Region,
		Latitude:  server.Latitude,
		Longitude: server.Longitude,
	}, nil
}

func (s *Servers) Servers(ctx context.Context, filter Query) ([]Server, error) {
	return s.repo.Query(ctx, filter)
}

// SeedRoleIDs returns the union of the discord seed role ids configured across all enabled,
// non-deleted servers. Duplicates are dropped, first seen wins.
func (s *Servers) SeedRoleIDs(ctx context.Context) ([]string, error) {
	srvs, errServers := s.Servers(ctx, Query{})
	if errServers != nil {
		return nil, errServers
	}

	var (
		seen    = make(map[string]struct{})
		roleIDs []string
	)

	for _, srv := range srvs {
		for _, roleID := range srv.DiscordSeedRoleIDs {
			if roleID == "" {
				continue
			}
			if _, dup := seen[roleID]; dup {
				continue
			}
			seen[roleID] = struct{}{}
			roleIDs = append(roleIDs, roleID)
		}
	}

	return roleIDs, nil
}

func (s *Servers) GetByName(ctx context.Context, serverName string) (Server, error) {
	server, errServer := s.repo.Query(ctx, Query{ShortName: serverName})

	if errServer != nil {
		return Server{}, errServer
	}

	if len(server) == 0 {
		return Server{}, ErrUnknownServer
	}

	return server[0], nil
}

func (s *Servers) GetByPassword(ctx context.Context, serverPassword string) (int32, string, error) {
	server, errServer := s.repo.Query(ctx, Query{Password: serverPassword, IncludeDisabled: true})
	if errServer != nil {
		return 0, "", errServer
	}

	if len(server) == 0 {
		return 0, "", ErrUnknownServer
	}

	return server[0].ServerID, server[0].ShortName, nil
}

func (s *Servers) Save(ctx context.Context, server Server) (Server, error) {
	if server.ServerID > 0 {
		server.UpdatedOn = time.Now()
	}

	if err := s.repo.Save(ctx, &server); err != nil {
		return Server{}, err
	}

	s.invalidateSecretCache()

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
		server.RLock()
		disabled := server.Deleted || !server.IsEnabled
		server.RUnlock()
		if disabled {
			continue
		}

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
