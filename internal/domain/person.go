package domain

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

type PersonProvider interface {
	// FIXME Retuning a interface for now.
	GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (PersonCore, error)
}

type DiscordPersonProvider interface {
	// FIXME Retuning a interface for now.
	GetPersonByDiscordID(ctx context.Context, discordID string) (PersonCore, error)
}

type PersonInfo interface {
	GetName() string
	GetAvatar() Avatar
	GetSteamID() steamid.SteamID
	Path() string // config.LinkablePath
	HasPermission(permission permission.Privilege) bool
	Permissions() permission.Privilege
}

// PersonCore is the model used in the webui representing the logged-in user.
type PersonCore struct {
	SteamID         steamid.SteamID      `json:"steam_id"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	Name            string               `json:"name"`
	Avatarhash      string               `json:"avatarhash"`
}

func (p PersonCore) Permissions() permission.Privilege {
	return p.PermissionLevel
}

func (p PersonCore) HasPermission(privilege permission.Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p PersonCore) GetName() string {
	if p.Name == "" {
		return p.SteamID.String()
	}

	return p.Name
}

func (p PersonCore) GetAvatar() Avatar {
	if p.Avatarhash == "" {
		return NewAvatar("fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb")
	}

	return NewAvatar(p.Avatarhash)
}

func (p PersonCore) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p PersonCore) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}
