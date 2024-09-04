package domain

import (
	"context"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/exp/slices"
	"net/url"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type NotificationRepository interface {
	SendSite(ctx context.Context, targetID steamid.Collection, severity NotificationSeverity, message string, link *url.URL) error
	GetPersonNotifications(ctx context.Context, filters NotificationQuery) ([]UserNotification, int64, error)
}

type NotificationUsecase interface {
	Enqueue(ctx context.Context, payload NotificationPayload)
	GetPersonNotifications(ctx context.Context, filters NotificationQuery) ([]UserNotification, int64, error)
	SendSite(ctx context.Context, recipients steamid.Collection, severity NotificationSeverity, message string, link *url.URL) error
	RegisterWorkers(workers *river.Workers)
	SetQueueClient(queueClient *river.Client[pgx.Tx])
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

type NotificationPayload struct {
	Types           []MessageType
	Sids            steamid.Collection
	Groups          []Privilege
	DiscordChannels []DiscordChannel
	Severity        NotificationSeverity
	Message         string
	DiscordEmbed    *discordgo.MessageEmbed
	Link            *url.URL
}

func (payload NotificationPayload) ValidationError() error {
	if slices.Contains(payload.Types, Discord) && len(payload.DiscordChannels) == 0 {
		return ErrDiscordChannelsEmpty
	}

	if slices.Contains(payload.Types, Discord) && payload.DiscordEmbed == nil {
		return ErrDiscordEmbedNil
	}

	if slices.Contains(payload.Types, User) && len(payload.Sids) == 0 {
		return ErrUserSteamIDsEmpty
	}

	return nil
}

func NewDiscordNotification(channel DiscordChannel, embed *discordgo.MessageEmbed) NotificationPayload {
	return NotificationPayload{
		Types:           []MessageType{Discord},
		Sids:            nil,
		Groups:          nil,
		DiscordChannels: []DiscordChannel{channel},
		Severity:        SeverityInfo,
		Message:         "",
		DiscordEmbed:    embed,
		Link:            nil,
	}
}

func NewSiteUserNotification(recipients steamid.Collection, severity NotificationSeverity, message string, link *url.URL) NotificationPayload {
	return NotificationPayload{
		Types:           []MessageType{User},
		Sids:            recipients,
		Groups:          nil,
		DiscordChannels: nil,
		Severity:        0,
		Message:         message,
		DiscordEmbed:    nil,
		Link:            link,
	}
}

func NewSiteGroupNotification(groups []Privilege, severity NotificationSeverity, message string, link *url.URL) NotificationPayload {
	return NotificationPayload{
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
