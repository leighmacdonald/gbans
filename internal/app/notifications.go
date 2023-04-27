package app

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/discordutil"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) sendNotification(notification model.NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= store.PUser {
		sids, errIds := app.store.GetSteamIdsAbove(app.ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}
		notification.Sids = append(notification.Sids, sids...)
	}
	uniqueIds := fp.Uniq(notification.Sids)
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
	go func(ids []string, pl model.NotificationPayload) {
		for _, discordId := range ids {
			embed := &discordgo.MessageEmbed{
				Title:       "Notification",
				Description: pl.Message,
			}
			if pl.Link != "" {
				embed.URL = config.ExtURL(pl.Link)
			}
			app.SendDiscordPayload(discordutil.Payload{ChannelId: discordId, Embed: embed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := app.store.SendNotification(app.ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			app.logger.Error("Failed to send notification", zap.Error(errSend))
			break
		}
	}
	return nil
}
