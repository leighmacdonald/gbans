package app

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NotificationPayload struct {
	MinPerms store.Privilege
	Sids     steamid.Collection
	Severity store.NotificationSeverity
	Message  string
	Link     string
}

func SendNotification(ctx context.Context, notification NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= store.PUser {
		sids, errIds := store.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}
		notification.Sids = append(notification.Sids, sids...)
	}
	uniqueIds := fp.Uniq(notification.Sids)
	people, errPeople := store.GetPeopleBySteamID(ctx, uniqueIds)
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
		for _, discordId := range ids {
			embed := &discordgo.MessageEmbed{
				Title:       "Notification",
				Description: pl.Message,
			}
			if pl.Link != "" {
				embed.URL = config.ExtURL(pl.Link)
			}
			discord.SendPayload(discord.Payload{ChannelId: discordId, Embed: embed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := store.SendNotification(ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			logger.Error("Failed to send notification", zap.Error(errSend))
			break
		}
	}
	return nil
}
