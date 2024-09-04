package notification

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
	"log/slog"
	"net/url"
)

func NewNotificationUsecase(repository domain.NotificationRepository, discord domain.DiscordUsecase) domain.NotificationUsecase {
	return &notificationUsecase{repository: repository, discord: discord}
}

type notificationUsecase struct {
	repository  domain.NotificationRepository
	discord     domain.DiscordUsecase
	queueClient *river.Client[pgx.Tx]
}

func (n *notificationUsecase) Enqueue(ctx context.Context, payload domain.NotificationPayload) {
	if n.queueClient == nil {
		return
	}

	_, err := n.queueClient.Insert(ctx, SenderArgs{Payload: payload}, &river.InsertOpts{Queue: string(queue.Default)})
	if err != nil {
		slog.Error("Failed to queue notification", log.ErrAttr(err))
	}
}

func (n *notificationUsecase) SendSite(ctx context.Context, targetIDs steamid.Collection, severity domain.NotificationSeverity, message string, link *url.URL) error {
	return n.repository.SendSite(ctx, fp.Uniq(targetIDs), severity, message, link)
}

func (n *notificationUsecase) SetQueueClient(queueClient *river.Client[pgx.Tx]) {
	n.queueClient = queueClient
}

func (n *notificationUsecase) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	return n.repository.GetPersonNotifications(ctx, filters)
}

func (n *notificationUsecase) RegisterWorkers(workers *river.Workers) {
	river.AddWorker[SenderArgs](workers, &SenderWorker{notifications: n})
}
