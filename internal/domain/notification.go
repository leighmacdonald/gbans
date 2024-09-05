package domain

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
	"golang.org/x/exp/slices"
)

type NotificationRepository interface {
	SendSite(ctx context.Context, targetID steamid.Collection, severity NotificationSeverity, message string, link string, authorID *int64) error
	GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]UserNotification, error)
	MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error
	MarkAllRead(ctx context.Context, steamID steamid.SteamID) error
	DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error
	DeleteAll(ctx context.Context, steamID steamid.SteamID) error
}

type NotificationUsecase interface {
	Enqueue(ctx context.Context, payload NotificationPayload)
	GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]UserNotification, error)
	SendSite(ctx context.Context, recipients steamid.Collection, severity NotificationSeverity, message string, link string, author *UserProfile) error
	RegisterWorkers(workers *river.Workers)
	SetQueueClient(queueClient *river.Client[pgx.Tx])
	MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error
	MarkAllRead(ctx context.Context, steamID steamid.SteamID) error
	DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error
	DeleteAll(ctx context.Context, steamID steamid.SteamID) error
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
	Link            string
	Author          *UserProfile
}

func (payload NotificationPayload) ValidationError() error {
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

func NewDiscordNotification(channel DiscordChannel, embed *discordgo.MessageEmbed) NotificationPayload {
	return NotificationPayload{
		Types:           []MessageType{Discord},
		Sids:            nil,
		Groups:          nil,
		DiscordChannels: []DiscordChannel{channel},
		Severity:        SeverityInfo,
		Message:         "",
		DiscordEmbed:    embed,
		Link:            "",
	}
}

func NewSiteUserNotification(recipients steamid.Collection, severity NotificationSeverity, message string, link string) NotificationPayload {
	return NotificationPayload{
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

func NewSiteUserNotificationWithAuthor(groups []Privilege, severity NotificationSeverity, message string, link string, author UserProfile) NotificationPayload {
	payload := NewSiteGroupNotification(groups, severity, message, link)
	payload.Author = &author

	return payload
}

func NewSiteGroupNotification(groups []Privilege, severity NotificationSeverity, message string, link string) NotificationPayload {
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

func NewSiteGroupNotificationWithAuthor(groups []Privilege, severity NotificationSeverity, message string, link string, author UserProfile) NotificationPayload {
	payload := NewSiteGroupNotification(groups, severity, message, link)
	payload.Author = &author

	return payload
}
