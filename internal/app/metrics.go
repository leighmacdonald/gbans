// Package app Package State is used for exporting State or other stats to prometheus.
package app

import (
	"context"

	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type metricCollector struct {
	logEventCounter     *prometheus.CounterVec
	sayCounter          *prometheus.CounterVec
	damageCounter       *prometheus.CounterVec
	healingCounter      *prometheus.CounterVec
	killCounter         *prometheus.CounterVec
	shotFiredCounter    *prometheus.CounterVec
	shotHitCounter      *prometheus.CounterVec
	mapCounter          *prometheus.CounterVec
	rconCounter         *prometheus.CounterVec
	connectedCounter    *prometheus.CounterVec
	disconnectedCounter *prometheus.CounterVec
	classCounter        *prometheus.CounterVec
	playerCounter       *prometheus.HistogramVec
}

func newMetricCollector() *metricCollector {
	collector := &metricCollector{
		logEventCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_log_events_total", Help: "Total log events ingested"},
			[]string{"server_name"}),

		sayCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_chat_total", Help: "Total chat messages sent"},
			[]string{"team_say"}),

		damageCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_damage_total", Help: "Total (real)damage dealt"},
			[]string{"weapon"}),

		healingCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_healing_total", Help: "Total (real)healing"},
			[]string{"weapon"}),

		killCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_kills_total", Help: "Total kills"},
			[]string{"weapon"}),

		shotFiredCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_fired_total", Help: "Total shots fired"},
			[]string{"weapon"}),

		shotHitCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_hit_total", Help: "Total shots hit"},
			[]string{"weapon"}),

		playerCounter: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "gbans_player_count", Help: "Players on a server"}, []string{"server_name"}),

		mapCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_map_played_total", Help: "Map played"},
			[]string{"map"}),

		rconCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_rcon_total", Help: "Total rcon commands executed"},
			[]string{"server_name"}),

		connectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_connected_total", Help: "Player connects"},
			[]string{"server_name"}),

		disconnectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_disconnected_total", Help: "Player disconnects"},
			[]string{"server_name"}),

		classCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_class_total", Help: "Player class"},
			[]string{"class"}),
	}
	for _, metric := range []prometheus.Collector{
		collector.damageCounter,
		collector.healingCounter,
		collector.killCounter,
		collector.shotFiredCounter,
		collector.shotHitCounter,
		collector.logEventCounter,
		collector.sayCounter,
		collector.playerCounter,
		collector.mapCounter,
		collector.rconCounter,
		collector.connectedCounter,
		collector.disconnectedCounter,
		collector.classCounter,
	} {
		_ = prometheus.Register(metric)
	}

	return collector
}

// logMetricsConsumer processes incoming log events and updated any associated metrics.
func logMetricsConsumer(ctx context.Context, collector *metricCollector, eb *fp.Broadcaster[logparse.EventType, logparse.ServerEvent], logger *zap.Logger) {
	log := logger.Named("metricsConsumer")

	eventChan := make(chan logparse.ServerEvent)
	if errRegister := eb.Consume(eventChan); errRegister != nil {
		log.Error("Failed to register event consumer", zap.Error(errRegister))

		return
	}

	parser := logparse.NewWeaponParser()

	for {
		select {
		case <-ctx.Done():
			return
		case newEvent := <-eventChan:
			if newEvent.ServerID == 0 {
				// TODO why is this ever nil?
				collector.logEventCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			}
			switch newEvent.EventType { //nolint:wsl,exhaustive
			case logparse.Damage:
				if evt, ok := newEvent.Event.(logparse.DamageEvt); ok {
					collector.damageCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Add(float64(evt.Damage))
				}
			case logparse.Healed:
				// evt := serverEvent.Event.(logparse.HealedEvt)
				// healingCounter.With(prometheus.Labels{"weapon": evt.Wa}).Add(float64(serverEvent.Damage))
			case logparse.ShotFired:
				if evt, ok := newEvent.Event.(logparse.ShotFiredEvt); ok {
					collector.shotFiredCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.ShotHit:
				if evt, ok := newEvent.Event.(logparse.ShotHitEvt); ok {
					collector.shotHitCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.Killed:
				if evt, ok := newEvent.Event.(logparse.KilledEvt); ok {
					collector.killCounter.With(prometheus.Labels{"weapon": parser.Name(evt.Weapon)}).Inc()
				}
			case logparse.Say:
				collector.sayCounter.With(prometheus.Labels{"team_say": "0"}).Inc()
			case logparse.SayTeam:
				collector.sayCounter.With(prometheus.Labels{"team_say": "1"}).Inc()
			case logparse.RCON:
				collector.rconCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.Connected:
				collector.connectedCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.Disconnected:
				collector.disconnectedCounter.With(prometheus.Labels{"server_name": newEvent.ServerName}).Inc()
			case logparse.SpawnedAs:
				if evt, ok := newEvent.Event.(logparse.SpawnedAsEvt); ok {
					collector.classCounter.With(prometheus.Labels{"class": evt.Class.String()}).Inc()
				}
			}
		}
	}
}
