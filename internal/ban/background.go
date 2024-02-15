package ban

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"go.uber.org/zap"
)

// Start periodically will query the database for expired bans and remove them.
func Start(ctx context.Context, logger *zap.Logger, banUsecase domain.BanSteamUsecase, banNetUsecase domain.BanNetUsecase,
	banASNUsecase domain.BanASNUsecase, personUsecase domain.PersonUsecase, discordUsecase domain.DiscordUsecase,
	configUsecase domain.ConfigUsecase,
) {
	var (
		log    = logger.Named("banSweeper")
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
					log.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						ban := expiredBan
						if errDrop := banUsecase.Delete(ctx, &ban, false); errDrop != nil {
							log.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							person, errPerson := personUsecase.GetPersonBySteamID(ctx, ban.TargetID)
							if errPerson != nil {
								log.Error("Failed to get expired Person", zap.Error(errPerson))

								continue
							}

							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}

							discordUsecase.SendPayload(domain.ChannelModLog, discord.BanExpiresMessage(ban, person, configUsecase.ExtURL(ban)))

							log.Info("Ban expired",
								zap.String("reason", ban.Reason.String()),
								zap.Int64("sid64", ban.TargetID.Int64()), zap.String("name", name))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := banNetUsecase.Expired(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, domain.ErrNoResult) {
					log.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := banNetUsecase.Delete(ctx, &expiredBan); errDropBanNet != nil {
							log.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							log.Info("IP ban expired", zap.String("cidr", expiredBan.String()))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := banASNUsecase.Expired(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, domain.ErrNoResult) {
					log.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						expired := expiredASNBan
						if errDropASN := banASNUsecase.Delete(ctx, &expired); errDropASN != nil {
							log.Error("Failed to drop expired asn ban", zap.Error(errDropASN))
						} else {
							log.Info("ASN ban expired", zap.Int64("ban_id", expired.BanASNId))
						}
					}
				}
			}()

			waitGroup.Wait()
		case <-ctx.Done():
			log.Debug("banSweeper shutting down")

			return
		}
	}
}
