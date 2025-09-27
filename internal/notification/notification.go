package notification

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/sliceutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type Notifier interface {
	Send(payload Payload)
}

type NullNotifier struct{}

func (n NullNotifier) Send(payload Payload) {}

type Severity int

const (
	Info Severity = iota
	Warn
	Error
)

type UserNotification struct {
	PersonNotificationID int64             `json:"person_notification_id"`
	SteamID              steamid.SteamID   `json:"steam_id"`
	Read                 bool              `json:"read"`
	Deleted              bool              `json:"deleted"`
	Severity             Severity          `json:"severity"`
	Message              string            `json:"message"`
	Link                 string            `json:"link"`
	Count                int               `json:"count"`
	Author               domain.PersonCore `json:"author"`
	CreatedOn            time.Time         `json:"created_on"`
}

type MessageType int

const (
	User MessageType = iota
	Discord
)

var (
	ErrUserSteamIDsEmpty    = errors.New("missing steam ids for recipients")
	ErrDiscordChannelsEmpty = errors.New("no channel ids provided")
	ErrDiscordEmbedNil      = errors.New("empty embed discord message provided")
)

type Payload struct {
	Types           []MessageType
	Sids            steamid.Collection
	Groups          []permission.Privilege
	DiscordChannels []string
	Severity        Severity
	Message         string
	DiscordEmbed    *discordgo.MessageEmbed
	Link            string
	Author          domain.PersonCore
}

func (payload Payload) ValidationError() error {
	if slices.Contains(payload.Types, Discord) && len(payload.DiscordChannels) == 0 {
		return ErrDiscordChannelsEmpty
	}

	if slices.Contains(payload.Types, Discord) && payload.DiscordEmbed == nil {
		return ErrDiscordEmbedNil
	}

	if slices.Contains(payload.Types, User) && len(payload.Sids) == 0 && len(payload.Groups) == 0 {
		return ErrUserSteamIDsEmpty
	}

	return nil
}

func NewDiscord(channel string, embed *discordgo.MessageEmbed) Payload {
	return Payload{
		Types:           []MessageType{Discord},
		Sids:            nil,
		Groups:          nil,
		DiscordChannels: []string{channel},
		Severity:        Info,
		Message:         "",
		DiscordEmbed:    embed,
		Link:            "",
	}
}

func NewSiteUser(recipients steamid.Collection, severity Severity, message string, link string) Payload {
	return Payload{
		Types:           []MessageType{User},
		Sids:            recipients,
		Groups:          nil,
		DiscordChannels: nil,
		Severity:        severity,
		Message:         message,
		DiscordEmbed:    nil,
		Link:            link,
	}
}

func NewSiteUserWithAuthor(groups []permission.Privilege, severity Severity, message string, link string, _ domain.PersonInfo) Payload {
	payload := NewSiteGroup(groups, severity, message, link)
	// payload.Author = &author

	return payload
}

func NewSiteGroup(groups []permission.Privilege, severity Severity, message string, link string) Payload {
	return Payload{
		Types:           []MessageType{User},
		Sids:            nil,
		Groups:          groups,
		DiscordChannels: nil,
		Severity:        severity,
		Message:         message,
		DiscordEmbed:    nil,
		Link:            link,
	}
}

func NewSiteGroupNotificationWithAuthor(groups []permission.Privilege, severity Severity, message string, link string, _ domain.PersonInfo) Payload {
	payload := NewSiteGroup(groups, severity, message, link)
	// payload.Author = &author

	return payload
}

func NewNotifications(repository Repository, discord *discord.Discord) *Notifications {
	return &Notifications{repository: repository, bot: discord}
}

type Notifications struct {
	repository Repository
	send       chan Payload
	bot        *discord.Discord
}

func (n *Notifications) Send(payload Payload) {
	n.send <- payload
}

func (n *Notifications) Sender(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notif := <-n.send:
			for _, channelID := range notif.DiscordChannels {
				n.bot.SendPayload(channelID, notif.DiscordEmbed)
			}
		}
	}
}

func (n *Notifications) SendSite(ctx context.Context, targetIDs steamid.Collection, severity Severity, message string, link string, author domain.PersonInfo) error {
	var authorID *int64
	sid := author.GetSteamID()
	if author != nil {
		sid64 := sid.Int64()
		authorID = &sid64
	}

	return n.repository.SendSite(ctx, sliceutil.Uniq(targetIDs), severity, message, link, authorID)
}

func (n *Notifications) GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]UserNotification, error) {
	return n.repository.GetPersonNotifications(ctx, steamID)
}

func (n *Notifications) MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.MarkMessagesRead(ctx, steamID, ids)
}

func (n *Notifications) MarkAllRead(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.MarkAllRead(ctx, steamID)
}

func (n *Notifications) DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return n.repository.DeleteMessages(ctx, steamID, ids)
}

func (n *Notifications) DeleteAll(ctx context.Context, steamID steamid.SteamID) error {
	return n.repository.DeleteAll(ctx, steamID)
}
