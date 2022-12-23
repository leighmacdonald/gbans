package app

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (app *App) sendNotification(notification notificationPayload) error {
	// Collect all required ids
	if notification.minPerms >= model.PUser {
		sids, errIds := app.store.GetSteamIdsAbove(app.ctx, notification.minPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}
		notification.sids = append(notification.sids, sids...)
	}
	uniqueIds := fp.Uniq(notification.sids)
	people, errPeople := app.store.GetPeopleBySteamID(app.ctx, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, store.ErrNoResult) {
		return errors.Wrap(errPeople, "Failed to fetch people for notification")
	}
	var discordIds []string
	for _, p := range people {
		if p.DiscordID != "" {
			discordIds = append(discordIds, p.DiscordID)
		}
	}
	go func(ids []string, pl notificationPayload) {
		for _, discordId := range ids {
			embed := &discordgo.MessageEmbed{
				Title:       "Notification",
				Description: pl.message,
			}
			if pl.link != "" {
				embed.URL = config.ExtURL(pl.link)
			}
			app.sendDiscordPayload(discordPayload{channelId: discordId, embed: embed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := app.store.SendNotification(app.ctx, sid, notification.severity,
			notification.message, notification.link); errSend != nil {
			log.WithError(errSend).Errorf("Failed to send notification")
			break
		}
	}
	return nil
}
