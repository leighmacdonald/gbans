package domain

import (
	"context"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

// todo add discord.
type NotificationRepository interface {
	SendNotification(ctx context.Context, targetID steamid.SteamID, severity NotificationSeverity, message string, link string) error
	GetPersonNotifications(ctx context.Context, filters NotificationQuery) ([]UserNotification, int64, error)
}
type NotificationUsecase interface {
	SendNotification(ctx context.Context, targetID steamid.SteamID, severity NotificationSeverity, message string, link string) error
	GetPersonNotifications(ctx context.Context, filters NotificationQuery) ([]UserNotification, int64, error)
}
type NotificationPayload struct {
	MinPerms Privilege
	Sids     steamid.Collection
	Severity NotificationSeverity
	Message  string
	Link     string
}
