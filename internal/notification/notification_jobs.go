package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/riverqueue/river"
)

type SenderArgs struct {
	Payload domain.NotificationPayload
}

func (args SenderArgs) Kind() string {
	return "notification"
}

func (args SenderArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      string(queue.Default),
		Priority:   int(queue.High),
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

func NewSenderWorker(people domain.PersonUsecase, notifications domain.NotificationUsecase, discord domain.DiscordUsecase) *SenderWorker {
	return &SenderWorker{
		people:        people,
		notifications: notifications,
		discord:       discord,
	}
}

type SenderWorker struct {
	river.WorkerDefaults[SenderArgs]
	discord       domain.DiscordUsecase
	people        domain.PersonUsecase
	notifications domain.NotificationUsecase
}

func (worker *SenderWorker) Work(ctx context.Context, job *river.Job[SenderArgs]) error {
	payload := job.Args.Payload

	if err := payload.ValidationError(); err != nil {
		return river.JobCancel(err)
	}

	worker.sendDiscord(payload)

	if err := worker.sendMessages(ctx, payload); err != nil {
		slog.Error("Error sending site messages", log.ErrAttr(err))

		return river.JobCancel(err)
	}

	return nil
}

func (worker *SenderWorker) sendDiscord(payload domain.NotificationPayload) {
	for _, channelID := range payload.DiscordChannels {
		worker.discord.SendPayload(channelID, payload.DiscordEmbed)
	}
}

func (worker *SenderWorker) sendMessages(ctx context.Context, payload domain.NotificationPayload) error {
	recipients := payload.Sids

	if len(payload.Groups) > 0 {
		groupRecipients, errGroups := worker.people.GetSteamIDsByGroups(ctx, payload.Groups)
		if errGroups != nil && !errors.Is(errGroups, domain.ErrNoResult) {
			return errGroups
		}

		recipients = append(recipients, groupRecipients...)
	}

	if len(recipients) > 0 {
		if errSend := worker.notifications.SendSite(ctx, recipients, payload.Severity, payload.Message, payload.Link); errSend != nil {
			slog.Error("Failed to send notification", log.ErrAttr(errSend))

			return errSend
		}
	}

	return nil
}
