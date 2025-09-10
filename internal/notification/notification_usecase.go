package notification

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

func NewNotificationUsecase(repository domain.NotificationRepository, discord discord.DiscordUsecase) domain.NotificationUsecase {
	return &notificationUsecase{repository: repository}
}

type notificationUsecase struct {
	repository  domain.NotificationRepository
	queueClient *river.Client[pgx.Tx]
}

func (n *notificationUsecase) Enqueue(ctx context.Context, payload domain.NotificationPayload) {
	if n.queueClient == nil {
		return
	}

	res, err := n.queueClient.Insert(ctx, SenderArgs{Payload: payload}, &river.InsertOpts{Queue: string(queue.Default)})
	if err != nil {
		slog.Error("Failed to queue notification", log.ErrAttr(err))
	}

	slog.Debug("Job inserted", slog.Int64("id", res.Job.ID), slog.Bool("unique", res.UniqueSkippedAsDuplicate))
}

func (n *notificationUsecase) SendSite(ctx context.Context, targetIDs steamid.Collection, severity domain.NotificationSeverity, message string, link string, author *domain.UserProfile) error {
	var authorID *int64
	if author != nil {
		sid64 := author.SteamID.Int64()
		authorID = &sid64
	}

	return n.repository.SendSite(ctx, fp.Uniq(targetIDs), severity, message, link, authorID)
}

func (n *notificationUsecase) SetQueueClient(queueClient *river.Client[pgx.Tx]) {
	n.queueClient = queueClient
}

func (n *notificationUsecase) GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]domain.UserNotification, error) {
	return n.repository.GetPersonNotifications(ctx, steamID)
}

func (n *notificationUsecase) RegisterWorkers(workers *river.Workers) {
	river.AddWorker[SenderArgs](workers, &SenderWorker{notifications: n})
}

func (n *notificationUsecase) MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.MarkMessagesRead(ctx, steamID, ids)
}

func (n *notificationUsecase) MarkAllRead(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.MarkAllRead(ctx, steamID)
}

func (n *notificationUsecase) DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.DeleteMessages(ctx, steamID, ids)
}

func (n *notificationUsecase) DeleteAll(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.DeleteAll(ctx, steamID)
}
