// Package app Package state is used for exporting state or other stats to prometheus.
package app

import (
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	logEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_log_events", Help: "Total log events ingested"},
		[]string{"server_name"})

	sayCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_chat", Help: "Total chat messages sent"},
		[]string{"team_say"})

	damageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_damage", Help: "Total (real)damage dealt"},
		[]string{"weapon"})

	healingCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_healing", Help: "Total (real)healing"},
		[]string{"weapon"})

	killCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_kills", Help: "Total kills"},
		[]string{"weapon"})

	shotFiredCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_shot_fired", Help: "Total shots fired"},
		[]string{"weapon"})

	shotHitCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_game_shot_hit", Help: "Total shots hit"},
		[]string{"weapon"})

	playerCounter = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "gbans_player_count", Help: "Players on a server"}, []string{"server_name"})

	mapCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_map_played", Help: "Map played"},
		[]string{"map"})

	rconCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_rcon", Help: "Total rcon commands executed"},
		[]string{"server_name"})

	connectedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_player_connected", Help: "Player connects"},
		[]string{"server_name"})

	disconnectedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_player_disconnected", Help: "Player disconnects"},
		[]string{"server_name"})

	classCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gbans_player_class", Help: "Player class"},
		[]string{"class"})
)

func init() {
	for _, m := range []prometheus.Collector{
		damageCounter,
		healingCounter,
		killCounter,
		shotFiredCounter,
		shotHitCounter,
		logEventCounter,
		sayCounter,
		playerCounter,
		mapCounter,
		rconCounter,
		connectedCounter,
		disconnectedCounter,
		classCounter,
	} {
		_ = prometheus.Register(m)
	}
}

// logMetricsConsumer processes incoming log events and updated any associated metrics
func logMetricsConsumer() {
	c := make(chan model.ServerEvent)
	if err := event.RegisterConsumer(c, []logparse.EventType{logparse.Any}); err != nil {
		log.Errorf("Failed to register event consumer")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c:
			if e.Server != nil {
				// TODO why is this ever nil?
				logEventCounter.With(prometheus.Labels{"server_name": e.Server.ServerName}).Inc()
			}
			switch e.EventType {
			case logparse.Damage:
				damageCounter.With(prometheus.Labels{"weapon": e.Weapon.String()}).Add(float64(e.Damage))
			case logparse.Healed:
				healingCounter.With(prometheus.Labels{"weapon": e.Weapon.String()}).Add(float64(e.Damage))
			case logparse.ShotFired:
				shotFiredCounter.With(prometheus.Labels{"weapon": e.Weapon.String()}).Inc()
			case logparse.ShotHit:
				shotHitCounter.With(prometheus.Labels{"weapon": e.Weapon.String()}).Inc()
			case logparse.Killed:
				killCounter.With(prometheus.Labels{"weapon": e.Weapon.String()}).Inc()
			case logparse.Say:
				sayCounter.With(prometheus.Labels{"team_say": "0"}).Inc()
			case logparse.SayTeam:
				sayCounter.With(prometheus.Labels{"team_say": "1"}).Inc()
			case logparse.RCON:
				rconCounter.With(prometheus.Labels{"server_name": e.Server.ServerName}).Inc()
			case logparse.Connected:
				connectedCounter.With(prometheus.Labels{"server_name": e.Server.ServerName}).Inc()
			case logparse.Disconnected:
				disconnectedCounter.With(prometheus.Labels{"server_name": e.Server.ServerName}).Inc()
			case logparse.SpawnedAs:
				classCounter.With(prometheus.Labels{"class": e.PlayerClass.String()}).Inc()
			}
		}
	}
}
