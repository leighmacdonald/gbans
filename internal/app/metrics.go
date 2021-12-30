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
		prometheus.CounterOpts{
			Name: "gbans_game_log_events",
			Help: "Total log events ingested",
		},
		[]string{"server_name"})

	sayCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_chat",
			Help: "Total chat messages sent",
		},
		[]string{"team_say"})

	damageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_damage",
			Help: "Total (real)damage dealt",
		},
		[]string{"weapon"})

	healingCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_healing",
			Help: "Total (real)healing",
		},
		[]string{"weapon"})

	killCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_kills",
			Help: "Total kills",
		},
		[]string{"weapon"})

	shotFiredCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_shot_fired",
			Help: "Total shots fired",
		},
		[]string{"weapon"})

	shotHitCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_shot_hit",
			Help: "Total shots hit",
		},
		[]string{"weapon"})
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
	} {
		_ = prometheus.Register(m)
	}
}

// logMetricsConsumer processes incoming log events and updated any associated metrics
func (g *gbans) logMetricsConsumer() {
	c := make(chan model.ServerEvent)
	if err := event.RegisterConsumer(c, []logparse.MsgType{logparse.Any}); err != nil {
		log.Errorf("Failed to register event consumer")
		return
	}
	for {
		select {
		case <-g.ctx.Done():
			return
		case e := <-c:
			logEventCounter.With(prometheus.Labels{"server_name": e.Server.ServerName}).Inc()
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
			}
		}
	}
}
