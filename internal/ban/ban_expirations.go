package ban

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
)

func NewExpirationMonitor(steam Bans, person person.Provider, notifications notification.Notifier, config *config.Configuration,
) *ExpirationMonitor {
	return &ExpirationMonitor{
		steam:         steam,
		person:        person,
		notifications: notifications,
		config:        config,
	}
}

type ExpirationMonitor struct {
	steam         Bans
	person        person.Provider
	notifications notification.Notifier
	config        *config.Configuration
}

func (monitor *ExpirationMonitor) Update(ctx context.Context) {
	expiredBans, errExpiredBans := monitor.steam.Expired(ctx)
	if errExpiredBans != nil && !errors.Is(errExpiredBans, database.ErrNoResult) {
		slog.Error("Failed to get expired expiredBans", slog.String("error", errExpiredBans.Error()))

		return
	}

	for _, expiredBan := range expiredBans {
		ban := expiredBan
		if errDrop := monitor.steam.Delete(ctx, &ban, false); errDrop != nil {
			slog.Error("Failed to drop expired expiredBan", slog.String("error", errDrop.Error()))

			continue
		}

		person, errPerson := monitor.person.GetOrCreatePersonBySteamID(ctx, ban.TargetID)
		if errPerson != nil {
			slog.Error("Failed to get expired Person", slog.String("error", errPerson.Error()))

			continue
		}

		name := person.Name
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
}
