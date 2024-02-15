package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type notificationUsecase struct {
	nr domain.NotificationRepository
	pu domain.PersonUsecase
}

func NewNotificationUsecase(repository domain.NotificationRepository,
	personUsecase domain.PersonUsecase,
) domain.NotificationUsecase {
	return &notificationUsecase{nr: repository, pu: personUsecase}
}

func (n notificationUsecase) SendNotification(ctx context.Context, targetID steamid.SID64, severity domain.NotificationSeverity, message string, link string) error {
	notification := domain.NotificationPayload{}
	// Collect all required ids
	if notification.MinPerms >= domain.PUser {
		sids, errIds := n.pu.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Join(errIds, domain.ErrNotificationSteamIDs)
		}

		notification.Sids = append(notification.Sids, sids...)
	}

	uniqueIds := fp.Uniq[steamid.SID64](notification.Sids)

	people, errPeople := n.pu.GetPeopleBySteamID(ctx, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, domain.ErrNoResult) {
		return errors.Join(errPeople, domain.ErrNotificationPeople)
	}

	var discordPeople []domain.Person

	for _, p := range people {
		if p.DiscordID != "" {
			discordPeople = append(discordPeople, p)
		}
	}

	go func(ids []domain.Person, payload domain.NotificationPayload) {
		for _, discordPerson := range discordPeople {
			if err := n.nr.SendNotification(ctx, discordPerson.SteamID, notification.Severity, notification.Message, notification.Link); err != nil {
				slog.Error("Failed to send discord notification", log.ErrAttr(err))
			}
		}
	}(discordPeople, notification)

	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := n.nr.SendNotification(ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			slog.Error("Failed to send notification", log.ErrAttr(errSend))

			break
		}
	}

	return n.nr.SendNotification(ctx, targetID, severity, message, link)
}

func (n notificationUsecase) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	return n.nr.GetPersonNotifications(ctx, filters)
}
