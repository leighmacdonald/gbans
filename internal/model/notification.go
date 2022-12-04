package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

type NotificationSeverity int

const (
	SeverityInfo NotificationSeverity = iota
	SeverityWarn
	SeverityError
)

type UserNotification struct {
	NotificationId int64                `json:"person_notification_id,string"`
	SteamId        steamid.SID64        `json:"steam_id,string"`
	Read           bool                 `json:"read"`
	Deleted        bool                 `json:"deleted"`
	Severity       NotificationSeverity `json:"severity"`
	Message        string               `json:"message"`
	Link           string               `json:"link"`
	Count          int                  `json:"count"`
	CreatedOn      time.Time            `json:"created_on"`
}
