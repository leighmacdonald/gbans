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
func Start(ctx context.Context, bansSteam domain.BanSteamUsecase, bansNet domain.BanNetUsecase,
	bansASN domain.BanASNUsecase, bansPerson domain.PersonUsecase, discordClient domain.DiscordUsecase,
	config domain.ConfigUsecase,
) {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)

			go func() {
				defer waitGroup.Done()

				expiredBans, errExpiredBans := bansSteam.Expired(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, domain.ErrNoResult) {
					slog.Error("Failed to get expired expiredBans", log.ErrAttr(errExpiredBans))

					return
				}

				for _, expiredBan := range expiredBans {
					ban := expiredBan
					if errDrop := bansSteam.Delete(ctx, &ban, false); errDrop != nil {
						slog.Error("Failed to drop expired expiredBan", log.ErrAttr(errDrop))

						continue
					}

					person, errPerson := bansPerson.GetPersonBySteamID(ctx, ban.TargetID)
					if errPerson != nil {
						slog.Error("Failed to get expired Person", log.ErrAttr(errPerson))

						continue
					}

					name := person.PersonaName
					if name == "" {
						name = person.SteamID.String()
					}

					discordClient.SendPayload(domain.ChannelBanLog, discord.BanExpiresMessage(ban, person, config.ExtURL(ban)))

					slog.Info("Ban expired",
						slog.String("reason", ban.Reason.String()),
						slog.Int64("sid64", ban.TargetID.Int64()), slog.String("name", name))
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := bansNet.Expired(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, domain.ErrNoResult) {
					slog.Warn("Failed to get expired network bans", log.ErrAttr(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := bansNet.Delete(ctx, expiredNetBan.NetID, domain.RequestUnban{UnbanReasonText: "Expired"}, false); errDropBanNet != nil {
							slog.Error("Failed to drop expired network expiredNetBan", log.ErrAttr(errDropBanNet))
						} else {
							slog.Info("IP ban expired", slog.String("cidr", expiredBan.String()), slog.Int64("net_id", expiredNetBan.NetID))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := bansASN.Expired(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, domain.ErrNoResult) {
					slog.Error("Failed to get expired asn bans", log.ErrAttr(errExpiredASNBans))
				} else {
					for _, expired := range expiredASNBans {
						if errDropASN := bansASN.Delete(ctx, expired.BanASNId, domain.RequestUnban{UnbanReasonText: "Expired"}); errDropASN != nil {
							slog.Error("Failed to drop expired asn ban", log.ErrAttr(errDropASN))
						} else {
							slog.Info("ASN ban expired", slog.Int64("ban_id", expired.BanASNId))
						}
					}
				}
			}()

			waitGroup.Wait()
		case <-ctx.Done():
			return
		}
	}
}
