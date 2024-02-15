package ban

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
)

// Start periodically will query the database for expired bans and remove them.
func Start(ctx context.Context, banUsecase domain.BanSteamUsecase, banNetUsecase domain.BanNetUsecase,
	banASNUsecase domain.BanASNUsecase, personUsecase domain.PersonUsecase, discordUsecase domain.DiscordUsecase,
	configUsecase domain.ConfigUsecase,
) {
	var (
		logger = slog.Default().WithGroup("banSweeper")
		ticker = time.NewTicker(time.Minute)
	)

	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)

			go func() {
				defer waitGroup.Done()

				expiredBans, errExpiredBans := banUsecase.Expired(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, domain.ErrNoResult) {
					logger.Error("Failed to get expired expiredBans", log.ErrAttr(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						ban := expiredBan
						if errDrop := banUsecase.Delete(ctx, &ban, false); errDrop != nil {
							logger.Error("Failed to drop expired expiredBan", log.ErrAttr(errDrop))
						} else {
							person, errPerson := personUsecase.GetPersonBySteamID(ctx, ban.TargetID)
							if errPerson != nil {
								logger.Error("Failed to get expired Person", log.ErrAttr(errPerson))

								continue
							}

							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}

							discordUsecase.SendPayload(domain.ChannelModLog, discord.BanExpiresMessage(ban, person, configUsecase.ExtURL(ban)))

							logger.Info("Ban expired",
								slog.String("reason", ban.Reason.String()),
								slog.Int64("sid64", ban.TargetID.Int64()), slog.String("name", name))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := banNetUsecase.Expired(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, domain.ErrNoResult) {
					logger.Warn("Failed to get expired network bans", log.ErrAttr(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := banNetUsecase.Delete(ctx, &expiredBan); errDropBanNet != nil {
							logger.Error("Failed to drop expired network expiredNetBan", log.ErrAttr(errDropBanNet))
						} else {
							logger.Info("IP ban expired", slog.String("cidr", expiredBan.String()))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := banASNUsecase.Expired(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, domain.ErrNoResult) {
					logger.Error("Failed to get expired asn bans", log.ErrAttr(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						expired := expiredASNBan
						if errDropASN := banASNUsecase.Delete(ctx, &expired); errDropASN != nil {
							logger.Error("Failed to drop expired asn ban", log.ErrAttr(errDropASN))
						} else {
							logger.Info("ASN ban expired", slog.Int64("ban_id", expired.BanASNId))
						}
					}
				}
			}()

			waitGroup.Wait()
		case <-ctx.Done():
			logger.Debug("banSweeper shutting down")

			return
		}
	}
}
