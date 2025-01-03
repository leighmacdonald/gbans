package ban

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewExpirationMonitor(steam domain.BanSteamUsecase, net domain.BanNetUsecase, asn domain.BanASNUsecase,
	person domain.PersonUsecase, notifications domain.NotificationUsecase, config domain.ConfigUsecase,
) *ExpirationMonitor {
	return &ExpirationMonitor{
		steam:         steam,
		net:           net,
		asn:           asn,
		person:        person,
		notifications: notifications,
		config:        config,
	}
}

type ExpirationMonitor struct {
	steam         domain.BanSteamUsecase
	net           domain.BanNetUsecase
	asn           domain.BanASNUsecase
	person        domain.PersonUsecase
	notifications domain.NotificationUsecase
	config        domain.ConfigUsecase
}

func (monitor *ExpirationMonitor) Update(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()

		expiredBans, errExpiredBans := monitor.steam.Expired(ctx)
		if errExpiredBans != nil && !errors.Is(errExpiredBans, domain.ErrNoResult) {
			slog.Error("Failed to get expired expiredBans", log.ErrAttr(errExpiredBans))

			return
		}

		for _, expiredBan := range expiredBans {
			ban := expiredBan
			if errDrop := monitor.steam.Delete(ctx, &ban, false); errDrop != nil {
				slog.Error("Failed to drop expired expiredBan", log.ErrAttr(errDrop))

				continue
			}

			person, errPerson := monitor.person.GetPersonBySteamID(ctx, nil, ban.TargetID)
			if errPerson != nil {
				slog.Error("Failed to get expired Person", log.ErrAttr(errPerson))

				continue
			}

			name := person.PersonaName
			if name == "" {
				name = person.SteamID.String()
			}

			monitor.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelBanLog, discord.BanExpiresMessage(ban, person, monitor.config.ExtURL(ban))))

			monitor.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
				[]steamid.SteamID{person.SteamID},
				domain.SeverityInfo,
				"Your mute/ban period has expired",
				ban.Path()))

			slog.Info("Ban expired",
				slog.String("reason", ban.Reason.String()),
				slog.Int64("sid64", ban.TargetID.Int64()), slog.String("name", name))
		}
	}()

	go func() {
		defer waitGroup.Done()

		expiredNetBans, errExpiredNetBans := monitor.net.Expired(ctx)
		if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, domain.ErrNoResult) {
			slog.Warn("Failed to get expired network bans", log.ErrAttr(errExpiredNetBans))
		} else {
			for _, expiredNetBan := range expiredNetBans {
				expiredBan := expiredNetBan
				if errDropBanNet := monitor.net.Delete(ctx, expiredNetBan.NetID, domain.RequestUnban{UnbanReasonText: "Expired"}, false); errDropBanNet != nil {
					if !errors.Is(errDropBanNet, domain.ErrNoResult) {
						slog.Error("Failed to drop expired network expiredNetBan", log.ErrAttr(errDropBanNet))
					}
				} else {
					slog.Info("IP ban expired", slog.String("cidr", expiredBan.String()), slog.Int64("net_id", expiredNetBan.NetID))
				}
			}
		}
	}()

	go func() {
		defer waitGroup.Done()

		expiredASNBans, errExpiredASNBans := monitor.asn.Expired(ctx)
		if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, domain.ErrNoResult) {
			slog.Error("Failed to get expired asn bans", log.ErrAttr(errExpiredASNBans))
		} else {
			for _, expired := range expiredASNBans {
				if errDropASN := monitor.asn.Delete(ctx, expired.BanASNId, domain.RequestUnban{UnbanReasonText: "Expired"}); errDropASN != nil {
					slog.Error("Failed to drop expired asn ban", log.ErrAttr(errDropASN))
				} else {
					slog.Info("ASN ban expired", slog.Int64("ban_id", expired.BanASNId))
				}
			}
		}
	}()

	waitGroup.Wait()
}
