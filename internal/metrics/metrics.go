package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// ServerSnapshot is the point-in-time server state used by the gauges. It
// keeps this package independent of the servers package.
type ServerSnapshot struct {
	Name   string
	Humans int32
}

// ServerProvider supplies a snapshot of the current servers at scrape time.
type ServerProvider func() []ServerSnapshot

// serverCollector produces server and player gauges from live state on each
// scrape, so the values never go stale between background updates.
type serverCollector struct {
	provider      ServerProvider
	serversTotal  *prometheus.Desc
	playersTotal  *prometheus.Desc
	serverPlayers *prometheus.Desc
}

func newServerCollector() *serverCollector {
	return &serverCollector{
		serversTotal:  prometheus.NewDesc("gbans_servers_total", "Number of enabled game servers", nil, nil),
		playersTotal:  prometheus.NewDesc("gbans_players_total", "Human players across all game servers", nil, nil),
		serverPlayers: prometheus.NewDesc("gbans_server_players", "Human players on a game server", []string{"server_name"}, nil),
	}
}

func (collector *serverCollector) Describe(descriptorChan chan<- *prometheus.Desc) {
	descriptorChan <- collector.serversTotal
	descriptorChan <- collector.playersTotal
	descriptorChan <- collector.serverPlayers
}

func (collector *serverCollector) Collect(metricChan chan<- prometheus.Metric) {
	snapshots := []ServerSnapshot{}
	if collector.provider != nil {
		snapshots = collector.provider()
	}

	var total float64
	for _, snapshot := range snapshots {
		players := float64(snapshot.Humans)
		total += players
		metricChan <- prometheus.MustNewConstMetric(collector.serverPlayers, prometheus.GaugeValue, players, snapshot.Name)
	}

	metricChan <- prometheus.MustNewConstMetric(collector.serversTotal, prometheus.GaugeValue, float64(len(snapshots)))
	metricChan <- prometheus.MustNewConstMetric(collector.playersTotal, prometheus.GaugeValue, total)
}

type collector struct {
	BuildInfo           *prometheus.GaugeVec
	LogEventCounter     prometheus.Counter
	SayCounter          *prometheus.CounterVec
	DamageCounter       *prometheus.CounterVec
	HealingCounter      *prometheus.CounterVec
	KillCounter         *prometheus.CounterVec
	ShotFiredCounter    *prometheus.CounterVec
	ShotHitCounter      *prometheus.CounterVec
	MapCounter          *prometheus.CounterVec
	RconCounter         *prometheus.CounterVec
	ConnectedCounter    *prometheus.CounterVec
	DisconnectedCounter *prometheus.CounterVec
	ClassCounter        *prometheus.CounterVec
	PlayerCounter       *prometheus.HistogramVec
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

type Metrics struct {
	registry        *prometheus.Registry
	collector       *collector
	serverCollector *serverCollector
	eb              *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func New(eventBus *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent], version string) (Metrics, error) {
	collectorSet := newMetricCollector()
	serverStats := newServerCollector()
	registry := prometheus.NewRegistry()

	registered := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		serverStats,
		collectorSet.BuildInfo,
		collectorSet.LogEventCounter,
		collectorSet.SayCounter,
		collectorSet.DamageCounter,
		collectorSet.HealingCounter,
		collectorSet.KillCounter,
		collectorSet.ShotFiredCounter,
		collectorSet.ShotHitCounter,
		collectorSet.MapCounter,
		collectorSet.RconCounter,
		collectorSet.ConnectedCounter,
		collectorSet.DisconnectedCounter,
		collectorSet.ClassCounter,
		collectorSet.PlayerCounter,
		collectorSet.HTTPRequestsTotal,
		collectorSet.HTTPRequestDuration,
	}
	for _, metric := range registered {
		if errRegister := registry.Register(metric); errRegister != nil {
			return Metrics{}, fmt.Errorf("register prometheus collector: %w", errRegister)
		}
	}

	collectorSet.BuildInfo.WithLabelValues(version).Set(1)

	return Metrics{registry: registry, collector: collectorSet, serverCollector: serverStats, eb: eventBus}, nil
}

// Registry exposes the private registry for the metrics HTTP handler.
func (u Metrics) Registry() *prometheus.Registry {
	return u.registry
}

// SetServerProvider connects the gauges to live server state.
func (u Metrics) SetServerProvider(provider ServerProvider) {
	u.serverCollector.provider = provider
}

// ObserveRequest records the outcome of a handled HTTP request.
func (u Metrics) ObserveRequest(method string, route string, status int, duration time.Duration) {
	u.collector.HTTPRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	u.collector.HTTPRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// Start begins processing incoming log events and updating any associated metrics.
func (u Metrics) Start(ctx context.Context) {
	eventChan := make(chan logparse.ServerEvent)
	if errRegister := u.eb.Consume(eventChan); errRegister != nil {
		slog.Error("Failed to register event consumer", slog.String("error", errRegister.Error()))

		return
	}

	parser := logparse.NewWeaponParser()

	for {
		select {
		case <-ctx.Done():
			return
		case newEvent := <-eventChan:
			u.collector.LogEventCounter.Inc()

			switch newEvent.EventType { //nolint:wsl,exhaustive
			case logparse.Damage:
				if evt, ok := newEvent.Event.(logparse.DamageEvt); ok {
					u.collector.DamageCounter.WithLabelValues(parser.Name(evt.Weapon)).Add(float64(evt.Damage))
				}
			case logparse.Healed:
				if evt, ok := newEvent.Event.(logparse.HealedEvt); ok {
					u.collector.HealingCounter.WithLabelValues("medigun").Add(float64(evt.Healing))
				}
			case logparse.ShotFired:
				if evt, ok := newEvent.Event.(logparse.ShotFiredEvt); ok {
					u.collector.ShotFiredCounter.WithLabelValues(parser.Name(evt.Weapon)).Inc()
				}
			case logparse.ShotHit:
				if evt, ok := newEvent.Event.(logparse.ShotHitEvt); ok {
					u.collector.ShotHitCounter.WithLabelValues(parser.Name(evt.Weapon)).Inc()
				}
			case logparse.Killed:
				if evt, ok := newEvent.Event.(logparse.KilledEvt); ok {
					u.collector.KillCounter.WithLabelValues(parser.Name(evt.Weapon)).Inc()
				}
			case logparse.Say:
				u.collector.SayCounter.With(prometheus.Labels{"team_say": "0"}).Inc()
			case logparse.SayTeam:
				u.collector.SayCounter.With(prometheus.Labels{"team_say": "1"}).Inc()
			case logparse.RCON:
				u.collector.RconCounter.WithLabelValues(newEvent.ServerName).Inc()
			case logparse.Connected:
				u.collector.ConnectedCounter.WithLabelValues(newEvent.ServerName).Inc()
			case logparse.Disconnected:
				u.collector.DisconnectedCounter.WithLabelValues(newEvent.ServerName).Inc()
			case logparse.SpawnedAs:
				if evt, ok := newEvent.Event.(logparse.SpawnedAsEvt); ok {
					u.collector.ClassCounter.With(prometheus.Labels{"class": evt.Class.String()}).Inc()
				}
			}
		}
	}
}

func newMetricCollector() *collector {
	return &collector{
		BuildInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "gbans_build_info", Help: "gbans build information"}, []string{"version"}),

		LogEventCounter: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "gbans_game_log_events_total", Help: "Total log events ingested"}),

		SayCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_chat_total", Help: "Total chat messages sent"},
			[]string{"team_say"}),

		DamageCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_damage_total", Help: "Total (real)damage dealt"},
			[]string{"weapon"}),

		HealingCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_healing_total", Help: "Total (real)healing"},
			[]string{"weapon"}),

		KillCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_kills_total", Help: "Total kills"},
			[]string{"weapon"}),

		ShotFiredCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_fired_total", Help: "Total shots fired"},
			[]string{"weapon"}),

		ShotHitCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_hit_total", Help: "Total shots hit"},
			[]string{"weapon"}),

		PlayerCounter: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "gbans_player_count", Help: "Players on a server"}, []string{"server_name"}),

		MapCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_map_played_total", Help: "Map played"},
			[]string{"map"}),

		RconCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_rcon_total", Help: "Total rcon commands executed"},
			[]string{"server_name"}),

		ConnectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_connected_total", Help: "Player connects"},
			[]string{"server_name"}),

		DisconnectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_disconnected_total", Help: "Player disconnects"},
			[]string{"server_name"}),

		ClassCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_class_total", Help: "Player class"},
			[]string{"class"}),

		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_http_requests_total", Help: "Total HTTP requests handled"},
			[]string{"method", "route", "code"}),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "gbans_http_request_duration_seconds", Help: "HTTP request duration in seconds"},
			[]string{"method", "route"}),
	}
}
