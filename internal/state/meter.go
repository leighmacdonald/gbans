package state

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	damageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_damage",
			Help: "Total (real)damage dealt",
		},
		[]string{"server_name", "steam_id", "target_id", "weapon"})
	healingCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_healing",
			Help: "Total (real)healing",
		},
		[]string{"server_name", "steam_id", "target_id", "healing"})
	killCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_kills",
			Help: "Total kills",
		},
		[]string{"server_name", "steam_id", "target_id", "weapon"})
	shotFiredCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_shot_fired",
			Help: "Total shots fired",
		},
		[]string{"server_name", "steam_id", "weapon"})
	shotHitCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gbans_game_shot_hit",
			Help: "Total shots hit",
		},
		[]string{"server_name", "steam_id", "weapon"})
)

func init() {
	for _, m := range []prometheus.Collector{
		damageCounter,
		healingCounter,
		killCounter,
		shotFiredCounter,
		shotHitCounter,
	} {
		_ = prometheus.Register(m)
	}
}

func LogMeter(ctx context.Context) {
	c := make(chan model.LogEvent)
	if err := event.RegisterConsumer(c, []logparse.MsgType{
		logparse.ShotHit,
		logparse.ShotFired,
		logparse.Damage,
		logparse.Killed,
		logparse.Healed,
	}); err != nil {
		log.Errorf("Failed to register event consumer")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c:
			switch e.Type {
			case logparse.Damage:
				var l logparse.DamageEvt
				if err := logparse.Unmarshal(e.Event, &l); err != nil {
					continue
				}
				d := l.Damage
				if l.RealDamage > 0 {
					d = l.RealDamage
				}
				damageCounter.With(prometheus.Labels{
					"server_name": e.Server.ServerName,
					"steam_id":    e.Player1.SteamID.String(),
					"target_id":   e.Player2.SteamID.String(),
					"weapon":      l.Weapon.String()}).
					Add(float64(d))
			case logparse.Healed:
				var l logparse.HealedEvt
				if err := logparse.Unmarshal(e.Event, &l); err != nil {
					continue
				}
				healingCounter.With(prometheus.Labels{
					"server_name": e.Server.ServerName,
					"steam_id":    e.Player1.SteamID.String(),
					"target_id":   e.Player2.SteamID.String()}).
					Add(float64(l.Healing))
			case logparse.ShotFired:
				var l logparse.ShotFiredEvt
				if err := logparse.Unmarshal(e.Event, &l); err != nil {
					continue
				}
				shotFiredCounter.With(prometheus.Labels{
					"server_name": e.Server.ServerName,
					"steam_id":    e.Player1.SteamID.String(),
					"weapon":      l.Weapon.String()}).
					Inc()
			case logparse.ShotHit:
				var l logparse.ShotHitEvt
				if err := logparse.Unmarshal(e.Event, &l); err != nil {
					continue
				}
				shotHitCounter.With(prometheus.Labels{
					"server_name": e.Server.ServerName,
					"steam_id":    e.Player1.SteamID.String(),
					"weapon":      l.Weapon.String()}).
					Inc()
			case logparse.Killed:
				var l logparse.KilledEvt
				if err := logparse.Unmarshal(e.Event, &l); err != nil {
					continue
				}
				killCounter.With(prometheus.Labels{
					"server_name": e.Server.ServerName,
					"steam_id":    e.Player1.SteamID.String(),
					"target_id":   e.Player2.SteamID.String(),
					"weapon":      l.Weapon.String()}).
					Inc()
			}

		}
	}
}
