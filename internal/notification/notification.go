package notification

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type NotificationSeverity int

const (
	SeverityInfo NotificationSeverity = iota
	SeverityWarn
	SeverityError
)

type NotificationQuery struct {
	domain.QueryFilter
	SteamID string `json:"steam_id"`
}

func (f NotificationQuery) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SteamID)

	return sid, sid.Valid()
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
	Groups          []permission.Privilege
	DiscordChannels []discord.DiscordChannel
	Severity        NotificationSeverity
	Message         string
	DiscordEmbed    *discordgo.MessageEmbed
	Link            string
	Author          *person.UserProfile
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

func NewDiscordNotification(channel discord.DiscordChannel, embed *discordgo.MessageEmbed) NotificationPayload {
	return NotificationPayload{
		Types:           []MessageType{Discord},
		Sids:            nil,
		Groups:          nil,
		DiscordChannels: []discord.DiscordChannel{channel},
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

func NewSiteUserNotificationWithAuthor(groups []permission.Privilege, severity NotificationSeverity, message string, link string, author domain.PersonInfo) NotificationPayload {
	payload := NewSiteGroupNotification(groups, severity, message, link)
	//payload.Author = &author

	return payload
}

func NewSiteGroupNotification(groups []permission.Privilege, severity NotificationSeverity, message string, link string) NotificationPayload {
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

func NewSiteGroupNotificationWithAuthor(groups []permission.Privilege, severity NotificationSeverity, message string, link string, author domain.PersonInfo) NotificationPayload {
	payload := NewSiteGroupNotification(groups, severity, message, link)
	//payload.Author = &author

	return payload
}
