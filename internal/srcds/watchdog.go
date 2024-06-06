package srcds

import (
	"context"
	"log/slog"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type activePlayer struct {
	created         time.Time
	lastUpdate      time.Time
	lastKickAttempt *time.Time
	banState        domain.PlayerBanState
}

type PlayerWatchdog struct {
	players map[steamid.SteamID]activePlayer
	state   domain.StateUsecase
	srcds   domain.SRCDSUsecase
}

func NewPlayerWatchdog(state domain.StateUsecase, srcds domain.SRCDSUsecase) PlayerWatchdog {
	return PlayerWatchdog{
		players: make(map[steamid.SteamID]activePlayer),
		state:   state,
		srcds:   srcds,
	}
}

func (w PlayerWatchdog) Start(ctx context.Context) {
	checkTicker := time.NewTicker(time.Second * 15)
	cleanupTicker := time.NewTicker(time.Minute)

	for {
		select {
		case <-cleanupTicker.C:
			w.cleanupExpired()
		case <-checkTicker.C:
			w.update(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w PlayerWatchdog) cleanupExpired() {
	for k := range w.players {
		if time.Since(w.players[k].lastUpdate) > time.Minute*5 {
			delete(w.players, k)
			slog.Debug("player expired", slog.String("sid", k.String()))
		}
	}
}

func (w PlayerWatchdog) update(ctx context.Context) {
	serverStates := w.state.Current()

	for _, server := range serverStates {
		for _, player := range server.Players {
			tracker, ok := w.players[player.SID]
			if !ok {
				tracker = activePlayer{
					created:    time.Now(),
					lastUpdate: time.Now(),
					banState:   domain.PlayerBanState{BanID: -1},
				}
			} else {
				tracker.lastUpdate = time.Now()
			}

			if tracker.banState.BanID < 0 {
				addr, errAddr := netip.ParseAddr(player.IP.String())
				if errAddr != nil {
					slog.Warn("Could not parse player ip", log.ErrAttr(errAddr))

					w.players[player.SID] = tracker

					continue
				}

				banState, _, err := w.srcds.GetBanState(ctx, player.SID, addr)
				if err != nil {
					w.players[player.SID] = tracker

					continue
				}

				tracker.banState = banState
			}

			if tracker.lastKickAttempt != nil && time.Since(*tracker.lastKickAttempt) > time.Second*60 && tracker.banState.BanType == domain.Banned {
				slog.Info("Kicking watchdog triggered player", slog.String("sid", player.SID.String()), slog.String("server", server.NameShort))

				if err := w.state.Kick(ctx, player.SID, tracker.banState.Reason); err != nil {
					slog.Error("Failed to kick watchdog triggered player", log.ErrAttr(err))
				}

				now := time.Now()
				tracker.lastKickAttempt = &now
			}

			w.players[player.SID] = tracker
		}
	}
}
