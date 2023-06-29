package app

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NotificationHandler struct{}

type NotificationPayload struct {
	MinPerms consts.Privilege
	Sids     steamid.Collection
	Severity consts.NotificationSeverity
	Message  string
	Link     string
}

func (app *App) SendNotification(ctx context.Context, notification NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= consts.PUser {
		sids, errIds := app.db.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}
		notification.Sids = append(notification.Sids, sids...)
	}
	uniqueIds := fp.Uniq(notification.Sids)
	people, errPeople := app.db.GetPeopleBySteamID(ctx, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, store.ErrNoResult) {
		return errors.Wrap(errPeople, "Failed to fetch people for notification")
	}
	var discordIds []string
	for _, p := range people {
		if p.DiscordID != "" {
			discordIds = append(discordIds, p.DiscordID)
		}
	}
	go func(ids []string, pl NotificationPayload) {
		for _, discordID := range ids {
			embed := &discordgo.MessageEmbed{
				Title:       "Notification",
				Description: pl.Message,
			}
			if pl.Link != "" {
				embed.URL = app.conf.ExtURL(pl.Link)
			}
			app.bot.SendPayload(discord.Payload{ChannelID: discordID, Embed: embed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := app.db.SendNotification(ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			app.log.Error("Failed to send notification", zap.Error(errSend))

			break
		}
	}

	return nil
}
