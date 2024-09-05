package ban

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type ExpirationArgs struct{}

func (args ExpirationArgs) Kind() string {
	return "bans_expired"
}

func (args ExpirationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: string(queue.Default), UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

func NewExpirationWorker(bansSteam domain.BanSteamUsecase, bansNet domain.BanNetUsecase, bansASN domain.BanASNUsecase,
	bansPerson domain.PersonUsecase, notifications domain.NotificationUsecase, config domain.ConfigUsecase,
) *ExpirationWorker {
	return &ExpirationWorker{
		bansSteam:     bansSteam,
		bansNet:       bansNet,
		bansASN:       bansASN,
		bansPerson:    bansPerson,
		notifications: notifications,
		config:        config,
	}
}

type ExpirationWorker struct {
	river.WorkerDefaults[ExpirationArgs]
	bansSteam     domain.BanSteamUsecase
	bansNet       domain.BanNetUsecase
	bansASN       domain.BanASNUsecase
	bansPerson    domain.PersonUsecase
	notifications domain.NotificationUsecase
	config        domain.ConfigUsecase
}

func (worker *ExpirationWorker) Work(ctx context.Context, _ *river.Job[ExpirationArgs]) error {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()

		expiredBans, errExpiredBans := worker.bansSteam.Expired(ctx)
		if errExpiredBans != nil && !errors.Is(errExpiredBans, domain.ErrNoResult) {
			slog.Error("Failed to get expired expiredBans", log.ErrAttr(errExpiredBans))

			return
		}

		for _, expiredBan := range expiredBans {
			ban := expiredBan
			if errDrop := worker.bansSteam.Delete(ctx, &ban, false); errDrop != nil {
				slog.Error("Failed to drop expired expiredBan", log.ErrAttr(errDrop))

				continue
			}

			person, errPerson := worker.bansPerson.GetPersonBySteamID(ctx, ban.TargetID)
			if errPerson != nil {
				slog.Error("Failed to get expired Person", log.ErrAttr(errPerson))

				continue
			}

			name := person.PersonaName
			if name == "" {
				name = person.SteamID.String()
			}

			worker.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelBanLog, discord.BanExpiresMessage(ban, person, worker.config.ExtURL(ban))))

			worker.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
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

		expiredNetBans, errExpiredNetBans := worker.bansNet.Expired(ctx)
		if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, domain.ErrNoResult) {
			slog.Warn("Failed to get expired network bans", log.ErrAttr(errExpiredNetBans))
		} else {
			for _, expiredNetBan := range expiredNetBans {
				expiredBan := expiredNetBan
				if errDropBanNet := worker.bansNet.Delete(ctx, expiredNetBan.NetID, domain.RequestUnban{UnbanReasonText: "Expired"}, false); errDropBanNet != nil {
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

		expiredASNBans, errExpiredASNBans := worker.bansASN.Expired(ctx)
		if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, domain.ErrNoResult) {
			slog.Error("Failed to get expired asn bans", log.ErrAttr(errExpiredASNBans))
		} else {
			for _, expired := range expiredASNBans {
				if errDropASN := worker.bansASN.Delete(ctx, expired.BanASNId, domain.RequestUnban{UnbanReasonText: "Expired"}); errDropASN != nil {
					slog.Error("Failed to drop expired asn ban", log.ErrAttr(errDropASN))
				} else {
					slog.Info("ASN ban expired", slog.Int64("ban_id", expired.BanASNId))
				}
			}
		}
	}()

	waitGroup.Wait()

	return nil
}
