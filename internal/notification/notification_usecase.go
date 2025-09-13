package notification

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewNotificationUsecase(repository NotificationRepository, discord *discord.Discord) NotificationUsecase {
	return NotificationUsecase{repository: repository}
}

type NotificationUsecase struct {
	repository NotificationRepository
}

func (n *NotificationUsecase) SendSite(ctx context.Context, targetIDs steamid.Collection, severity NotificationSeverity, message string, link string, author domain.PersonInfo) error {
	var authorID *int64
	sid := author.GetSteamID()
	if author != nil {
		sid64 := sid.Int64()
		authorID = &sid64
	}

	return n.repository.SendSite(ctx, fp.Uniq(targetIDs), severity, message, link, authorID)
}

func (n *NotificationUsecase) GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]UserNotification, error) {
	return n.repository.GetPersonNotifications(ctx, steamID)
}

func (n *NotificationUsecase) MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.MarkMessagesRead(ctx, steamID, ids)
}

func (n *NotificationUsecase) MarkAllRead(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.MarkAllRead(ctx, steamID)
}

func (n *NotificationUsecase) DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.DeleteMessages(ctx, steamID, ids)
}

func (n *NotificationUsecase) DeleteAll(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.DeleteAll(ctx, steamID)
}
