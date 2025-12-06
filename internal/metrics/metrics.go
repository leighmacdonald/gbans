package metrics

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	LogEventCounter     *prometheus.CounterVec
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
}

type Metrics struct {
	collector *collector
	eb        *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func New(broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]) Metrics {
	collector := newMetricCollector()

	return Metrics{collector: collector, eb: broadcaster}
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
			// if newEvent.ServerID == 0 {
			// TODO why is this ever nil?
			// u.collector.LogEventCounter.With(prometheus.Labels{"server_name": newEvent.ServerID}).Inc()
			// }
			switch newEvent.EventType { //nolint:wsl,exhaustive
			case logparse.Damage:
				if evt, ok := newEvent.Event.(logparse.DamageEvt); ok {
					u.collector.DamageCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Add(float64(evt.Damage))
				}
			case logparse.Healed:
				// evt := serverEvent.Event.(logparse.HealedEvt)
				// healingCounter.With(prometheus.Labels{"weapon": evt.Wa}).Add(float64(serverEvent.Damage))
			case logparse.ShotFired:
				if evt, ok := newEvent.Event.(logparse.ShotFiredEvt); ok {
					u.collector.ShotFiredCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.ShotHit:
				if evt, ok := newEvent.Event.(logparse.ShotHitEvt); ok {
					u.collector.ShotHitCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.Killed:
				if evt, ok := newEvent.Event.(logparse.KilledEvt); ok {
					u.collector.KillCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.Say:
				u.collector.SayCounter.With(prometheus.Labels{"team_say": "0"}).Inc()
			case logparse.SayTeam:
				u.collector.SayCounter.With(prometheus.Labels{"team_say": "1"}).Inc()
			case logparse.RCON:
				// u.collector.RconCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.Connected:
				// u.collector.ConnectedCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.Disconnected:
				// u.collector.DisconnectedCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.SpawnedAs:
				if evt, ok := newEvent.Event.(logparse.SpawnedAsEvt); ok {
					u.collector.ClassCounter.With(prometheus.Labels{"class": evt.Class.String()}).Inc()
				}
			}
		}
	}
}

func newMetricCollector() *collector {
	collector := &collector{
		LogEventCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_log_events_total", Help: "Total log events ingested"},
			[]string{"server_name"}),

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
	}
	for _, metric := range []prometheus.Collector{
		collector.DamageCounter,
		collector.HealingCounter,
		collector.KillCounter,
		collector.ShotFiredCounter,
		collector.ShotHitCounter,
		collector.LogEventCounter,
		collector.SayCounter,
		collector.PlayerCounter,
		collector.MapCounter,
		collector.RconCounter,
		collector.ConnectedCounter,
		collector.DisconnectedCounter,
		collector.ClassCounter,
	} {
		_ = prometheus.Register(metric)
	}

	return collector
}
