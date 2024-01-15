package common

import "github.com/leighmacdonald/steamid/v3/steamid"

type PersonInfo interface {
	GetDiscordID() string
	GetName() string
	GetAvatar() AvatarLinks
	GetSteamID() steamid.SID64
	Path() string // config.LinkablePath
}
