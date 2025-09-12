package domain

import (
	"fmt"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	avatarURLSmallFormat  = "https://avatars.akamai.steamstatic.com/%s.jpg"
	avatarURLMediumFormat = "https://avatars.akamai.steamstatic.com/%s_medium.jpg"
	avatarURLFullFormat   = "https://avatars.akamai.steamstatic.com/%s_full.jpg"
)

func NewAvatar(hash string) Avatar {
	return Avatar{hash: hash}
}

type Avatar struct {
	hash string
}

func (h Avatar) Full() string {
	return fmt.Sprintf(avatarURLFullFormat, h.hash)
}

func (h Avatar) Medium() string {
	return fmt.Sprintf(avatarURLMediumFormat, h.hash)
}

func (h Avatar) Small() string {
	return fmt.Sprintf(avatarURLSmallFormat, h.hash)
}

func (h Avatar) Hash() string {
	return h.hash
}

type PersonInfo interface {
	GetDiscordID() string
	GetName() string
	GetAvatar() Avatar
	GetSteamID() steamid.SteamID
	Path() string // config.LinkablePath
	HasPermission(permission permission.Privilege) bool
	Permissions() permission.Privilege
}
