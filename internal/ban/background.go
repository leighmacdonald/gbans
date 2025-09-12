package ban

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/pkg/log"
)

func NewExpirationMonitor(steam BanUsecase, person *person.PersonUsecase, notifications notification.NotificationUsecase, config *config.ConfigUsecase,
) *ExpirationMonitor {
	return &ExpirationMonitor{
		steam:         steam,
		person:        person,
		notifications: notifications,
		config:        config,
	}
}

type ExpirationMonitor struct {
	steam         BanUsecase
	person        *person.PersonUsecase
	notifications notification.NotificationUsecase
	config        *config.ConfigUsecase
}

func (monitor *ExpirationMonitor) Update(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		expiredBans, errExpiredBans := monitor.steam.Expired(ctx)
		if errExpiredBans != nil && !errors.Is(errExpiredBans, database.ErrNoResult) {
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

			// monitor.notifications.Enqueue(ctx, notification.NewDiscordNotification(monitor.config.Config().Discord.BanLogChannelID, discord.BanExpiresMessage(ban, person, monitor.config.ExtURL(ban))))

			// monitor.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
			// 	[]steamid.SteamID{person.SteamID},
			// 	notification.SeverityInfo,
			// 	"Your mute/ban period has expired",
			// 	ban.Path()))

			slog.Info("Ban expired",
				slog.String("reason", ban.Reason.String()),
				slog.Int64("sid64", ban.TargetID.Int64()), slog.String("name", name))
		}
	}()

	waitGroup.Wait()
}
