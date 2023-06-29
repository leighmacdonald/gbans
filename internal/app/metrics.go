// Package app Package state is used for exporting state or other stats to prometheus.
package app

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/model"
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
	mc := &metricCollector{
		logEventCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_log_events", Help: "Total log events ingested"},
			[]string{"server_name"}),

		sayCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_chat", Help: "Total chat messages sent"},
			[]string{"team_say"}),

		damageCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_damage", Help: "Total (real)damage dealt"},
			[]string{"weapon"}),

		healingCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_healing", Help: "Total (real)healing"},
			[]string{"weapon"}),

		killCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_kills", Help: "Total kills"},
			[]string{"weapon"}),

		shotFiredCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_fired", Help: "Total shots fired"},
			[]string{"weapon"}),

		shotHitCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_game_shot_hit", Help: "Total shots hit"},
			[]string{"weapon"}),

		playerCounter: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "gbans_player_count", Help: "Players on a server"}, []string{"server_name"}),

		mapCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_map_played", Help: "Map played"},
			[]string{"map"}),

		rconCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_rcon", Help: "Total rcon commands executed"},
			[]string{"server_name"}),

		connectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_connected", Help: "Player connects"},
			[]string{"server_name"}),

		disconnectedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_disconnected", Help: "Player disconnects"},
			[]string{"server_name"}),

		classCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "gbans_player_class", Help: "Player class"},
			[]string{"class"}),
	}
	for _, m := range []prometheus.Collector{
		mc.damageCounter,
		mc.healingCounter,
		mc.killCounter,
		mc.shotFiredCounter,
		mc.shotHitCounter,
		mc.logEventCounter,
		mc.sayCounter,
		mc.playerCounter,
		mc.mapCounter,
		mc.rconCounter,
		mc.connectedCounter,
		mc.disconnectedCounter,
		mc.classCounter,
	} {
		_ = prometheus.Register(m)
	}
	return mc
}

// logMetricsConsumer processes incoming log events and updated any associated metrics.
func logMetricsConsumer(ctx context.Context, mc *metricCollector, eb *eventBroadcaster, logger *zap.Logger) {
	log := logger.Named("metricsConsumer")
	eventChan := make(chan model.ServerEvent)
	if errRegister := eb.Consume(eventChan, []logparse.EventType{logparse.Any}); errRegister != nil {
		log.Error("Failed to register event consumer", zap.Error(errRegister))
		return
	}
	parser := logparse.New()
	for {
		select {
		case <-ctx.Done():
			return
		case serverEvent := <-eventChan:
			if serverEvent.Server.ServerID == 0 {
				// TODO why is this ever nil?
				mc.logEventCounter.With(prometheus.Labels{"server_name": serverEvent.Server.ServerNameShort}).Inc()
			}
			switch serverEvent.EventType { //nolint:wsl,exhaustive
			case logparse.Damage:
				if evt, ok := serverEvent.Event.(logparse.DamageEvt); ok {
					mc.damageCounter.With(prometheus.Labels{"weapon": parser.WeaponName(evt.Weapon)}).Add(float64(evt.Damage))
				}
			case logparse.Healed:
				// evt := serverEvent.Event.(logparse.HealedEvt)
				// healingCounter.With(prometheus.Labels{"weapon": evt.Wa}).Add(float64(serverEvent.Damage))
			case logparse.ShotFired:
				if evt, ok := serverEvent.Event.(logparse.ShotFiredEvt); ok {
					mc.shotFiredCounter.With(prometheus.Labels{"weapon": parser.WeaponName(evt.Weapon)}).Inc()
				}
			case logparse.ShotHit:
				if evt, ok := serverEvent.Event.(logparse.ShotHitEvt); ok {
					mc.shotHitCounter.With(prometheus.Labels{"weapon": parser.WeaponName(evt.Weapon)}).Inc()
				}
			case logparse.Killed:
				if evt, ok := serverEvent.Event.(logparse.KilledEvt); ok {
					mc.killCounter.With(prometheus.Labels{"weapon": parser.WeaponName(evt.Weapon)}).Inc()
				}
			case logparse.Say:
				mc.sayCounter.With(prometheus.Labels{"team_say": "0"}).Inc()
			case logparse.SayTeam:
				mc.sayCounter.With(prometheus.Labels{"team_say": "1"}).Inc()
			case logparse.RCON:
				mc.rconCounter.With(prometheus.Labels{"server_name": serverEvent.Server.ServerNameShort}).Inc()
			case logparse.Connected:
				mc.connectedCounter.With(prometheus.Labels{"server_name": serverEvent.Server.ServerNameShort}).Inc()
			case logparse.Disconnected:
				mc.disconnectedCounter.With(prometheus.Labels{"server_name": serverEvent.Server.ServerNameShort}).Inc()
			case logparse.SpawnedAs:
				if evt, ok := serverEvent.Event.(logparse.SpawnedAsEvt); ok {
					mc.classCounter.With(prometheus.Labels{"class": evt.PlayerClass.String()}).Inc()
				}
			}
		}
	}
}
