package health

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	v1 "github.com/leighmacdonald/gbans/internal/health/v1"
	"github.com/leighmacdonald/gbans/internal/health/v1/healthv1connect"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config holds the static settings the health checks need. Paths are optional;
// empty entries are skipped.
type Config struct {
	Version           string
	AssetRoot         string
	GeoIPCache        string
	CleanupMount      string
	DiscordEnabled    bool
	SSHEnabled        bool
	PrometheusEnabled bool
}

type Service struct {
	healthv1connect.UnimplementedHealthServiceHandler

	tracker        *Tracker
	database       database.Database
	bot            discord.Connection
	config         Config
	serverProvider func() []ServerSnapshot
}

func NewService(
	tracker *Tracker,
	database database.Database,
	bot discord.Connection,
	conf Config,
	serverProvider func() []ServerSnapshot,
	openGraph *opengraph.Repository,
	roleAuth *rpc.RoleAuth,
	authMiddleware *rpc.Middleware,
	option ...connect.HandlerOption,
) rpc.Service {
	pattern, handler := healthv1connect.NewHealthServiceHandler(Service{
		tracker:        tracker,
		database:       database,
		bot:            bot,
		config:         conf,
		serverProvider: serverProvider,
	}, option...)

	openGraph.Register("/admin/health", opengraph.Static("System Health", "Service and server health overview."))
	authMiddleware.UserRoute(healthv1connect.HealthServiceHealthProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_READ))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

type check struct {
	name   string
	status v1.HealthStatus
	detail string
	at     time.Time
}

func newItem(result check) *v1.HealthItem {
	item := &v1.HealthItem{
		Name:   &result.name,
		Status: &result.status,
		Detail: &result.detail,
	}
	if !result.at.IsZero() {
		item.LastCheck = timestamppb.New(result.at)
	}

	return item
}

func diskCheck(name string, root string) check {
	if root == "" {
		return check{name: name, status: v1.HealthStatus_HEALTH_STATUS_DISABLED, detail: "not configured"}
	}

	total, free, errStat := diskStats(root)
	if errStat != nil {
		return check{
			name: name, status: v1.HealthStatus_HEALTH_STATUS_ERROR,
			detail: fmt.Sprintf("cannot stat %s: %v", root, errStat), at: time.Now(),
		}
	}

	if total == 0 {
		return check{name: name, status: v1.HealthStatus_HEALTH_STATUS_ERROR, detail: "zero-size filesystem", at: time.Now()}
	}

	usedPct := float64(total-free) / float64(total) * 100
	status := v1.HealthStatus_HEALTH_STATUS_OK
	if usedPct >= 95 {
		status = v1.HealthStatus_HEALTH_STATUS_ERROR
	} else if usedPct >= 80 {
		status = v1.HealthStatus_HEALTH_STATUS_WARN
	}

	return check{
		name:   name,
		status: status,
		detail: fmt.Sprintf("%s: %.1f%% used, %s free", root, usedPct, humanBytes(free)),
		at:     time.Now(),
	}
}

func humanBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	for _, suffix := range units {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}

	return fmt.Sprintf("%.1f PiB", value/unit)
}

func (s Service) databaseCheck(ctx context.Context) check {
	pool := s.database.Pool()
	if pool == nil {
		return check{name: "Database", status: v1.HealthStatus_HEALTH_STATUS_ERROR, detail: "no connection pool"}
	}

	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	status := v1.HealthStatus_HEALTH_STATUS_OK
	detail := ""
	if errPing := pool.Ping(pingCtx); errPing != nil {
		status = v1.HealthStatus_HEALTH_STATUS_ERROR
		detail = fmt.Sprintf("ping failed: %v; ", errPing)
	}
	stats := pool.Stat()

	return check{
		name:   "Database",
		status: status,
		detail: fmt.Sprintf("%s%d/%d connections, %d idle, ping %s", detail,
			stats.AcquiredConns(), stats.MaxConns(), stats.IdleConns(), time.Since(start).Round(time.Millisecond)),
		at: time.Now(),
	}
}

func (s Service) discordCheck() check {
	if !s.config.DiscordEnabled {
		return check{name: "Discord Bot", status: v1.HealthStatus_HEALTH_STATUS_DISABLED, detail: "disabled"}
	}
	if s.bot == nil || !s.bot.IsReady() {
		return check{
			name: "Discord Bot", status: v1.HealthStatus_HEALTH_STATUS_WARN,
			detail: "configured but not connected", at: time.Now(),
		}
	}

	return check{name: "Discord Bot", status: v1.HealthStatus_HEALTH_STATUS_OK, detail: "connected", at: time.Now()}
}

func newestEvent(events map[string]Event, names ...string) Event {
	var newest Event
	for _, name := range names {
		if event, ok := events[name]; ok && event.At.After(newest.At) {
			newest = event
		}
	}

	return newest
}

func eventCheck(name string, event Event, staleAfter time.Duration) check {
	if event.At.IsZero() {
		return check{name: name, status: v1.HealthStatus_HEALTH_STATUS_WARN, detail: "no activity recorded"}
	}
	if event.Err != "" {
		return check{name: name, status: v1.HealthStatus_HEALTH_STATUS_ERROR, detail: event.Err, at: event.At}
	}
	if staleAfter > 0 && time.Since(event.At) > staleAfter {
		return check{
			name: name, status: v1.HealthStatus_HEALTH_STATUS_WARN,
			detail: fmt.Sprintf("last success %s ago", time.Since(event.At).Round(time.Second)), at: event.At,
		}
	}

	return check{
		name: name, status: v1.HealthStatus_HEALTH_STATUS_OK,
		detail: fmt.Sprintf("last success %s ago", time.Since(event.At).Round(time.Second)), at: event.At,
	}
}

func (s Service) rconCheck(snapshot Snapshot) check {
	var successes, failures int
	var oldest time.Time
	for _, event := range snapshot.RCON {
		if event.Err != "" {
			failures++

			continue
		}
		successes++
		if oldest.IsZero() || event.At.Before(oldest) {
			oldest = event.At
		}
	}

	detail := fmt.Sprintf("%d servers successful, %d failed", successes, failures)
	if !oldest.IsZero() {
		detail += fmt.Sprintf(", oldest success %s ago", time.Since(oldest).Round(time.Second))
	}

	status := v1.HealthStatus_HEALTH_STATUS_OK
	if successes == 0 {
		status = v1.HealthStatus_HEALTH_STATUS_WARN
	} else if failures > 0 {
		status = v1.HealthStatus_HEALTH_STATUS_WARN
	}

	return check{name: "RCON", status: status, detail: detail, at: oldest}
}

func (s Service) sshCheck(snapshot Snapshot) check {
	if !s.config.SSHEnabled {
		return check{name: "SSH", status: v1.HealthStatus_HEALTH_STATUS_DISABLED, detail: "disabled"}
	}

	var newest Event
	for _, event := range snapshot.SSH {
		if event.At.After(newest.At) {
			newest = event
		}
	}

	return eventCheck("SSH", newest, 2*time.Hour)
}

func (s Service) serverCheck(servers []ServerSnapshot) check {
	if len(servers) == 0 {
		return check{name: "Game Servers", status: v1.HealthStatus_HEALTH_STATUS_WARN, detail: "no enabled servers"}
	}

	var stale, never, withSTV int
	for _, server := range servers {
		if server.STVPort > 0 {
			withSTV++
		}
		if server.LastStatusUpdate.IsZero() {
			never++

			continue
		}
		if time.Since(server.LastStatusUpdate) > time.Minute {
			stale++
		}
	}

	detail := fmt.Sprintf("%d servers, %d with SourceTV, %d stale, %d never polled", len(servers), withSTV, stale, never)
	status := v1.HealthStatus_HEALTH_STATUS_OK
	if stale > 0 || never > 0 {
		status = v1.HealthStatus_HEALTH_STATUS_WARN
	}
	if stale == len(servers) {
		status = v1.HealthStatus_HEALTH_STATUS_ERROR
	}

	return check{name: "Game Servers", status: status, detail: detail, at: time.Now()}
}

func (s Service) Health(ctx context.Context, _ *emptypb.Empty) (*v1.HealthResponse, error) {
	snapshot := s.tracker.Snapshot()
	servers := []ServerSnapshot{}
	if s.serverProvider != nil {
		servers = s.serverProvider()
		sort.Slice(servers, func(i, j int) bool { return servers[i].ServerID < servers[j].ServerID })
	}

	checks := []check{
		s.databaseCheck(ctx),
		s.discordCheck(),
		diskCheck("Disk (assets)", s.config.AssetRoot),
		diskCheck("Disk (GeoIP)", s.config.GeoIPCache),
		diskCheck("Disk (demos)", s.config.CleanupMount),
		s.sshCheck(snapshot),
		s.rconCheck(snapshot),
		eventCheck("Stats Refresh", newestEvent(snapshot.Events, "stats_refresh"), 2*time.Hour),
		eventCheck("GeoIP Update", newestEvent(snapshot.Events, "geoip"), 48*time.Hour),
		eventCheck("Background Workers", newestEvent(snapshot.Events, "worker_tick"), 5*time.Minute),
		s.serverCheck(servers),
		{
			name:   "Uptime",
			status: v1.HealthStatus_HEALTH_STATUS_OK,
			detail: fmt.Sprintf("%s, build %s", time.Since(snapshot.Started).Round(time.Second), s.config.Version),
			at:     snapshot.Started,
		},
	}

	resp := &v1.HealthResponse{CheckedOn: timestamppb.Now()}
	for _, result := range checks {
		resp.Items = append(resp.Items, newItem(result))
	}

	for _, server := range servers {
		name := server.Name
		id := server.ServerID
		enabled := server.Enabled
		stvPort := uint32(server.STVPort) //nolint:gosec
		row := &v1.ServerHealth{
			ServerId: &id,
			Name:     &name,
			StvPort:  &stvPort,
			Enabled:  &enabled,
		}
		if event, ok := snapshot.RCON[server.ServerID]; ok && event.Err == "" {
			row.LastRconSuccess = timestamppb.New(event.At)
		}
		if !server.LastStatusUpdate.IsZero() {
			row.LastStatusUpdate = timestamppb.New(server.LastStatusUpdate)
		}
		if !server.TokenCreatedOn.IsZero() {
			row.LastAuth = timestamppb.New(server.TokenCreatedOn)
		}
		resp.Servers = append(resp.Servers, row)
	}

	return resp, nil
}

// String is used by tests and logs to describe a health status succinctly.
func StatusString(status v1.HealthStatus) string {
	return strings.ToLower(strings.TrimPrefix(status.String(), "HEALTH_STATUS_"))
}
