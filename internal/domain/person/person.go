package person

import (
	"context"
	"fmt"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	avatarURLSmallFormat  = "https://avatars.akamai.steamstatic.com/%s.jpg"
	avatarURLMediumFormat = "https://avatars.akamai.steamstatic.com/%s_medium.jpg"
	avatarURLFullFormat   = "https://avatars.akamai.steamstatic.com/%s_full.jpg"
)

type Avatar string

func (h Avatar) Full() string {
	return fmt.Sprintf(avatarURLFullFormat, h)
}

func (h Avatar) Medium() string {
	return fmt.Sprintf(avatarURLMediumFormat, h)
}

func (h Avatar) Small() string {
	return fmt.Sprintf(avatarURLSmallFormat, h)
}

func (h Avatar) Hash() string {
	return string(h)
}

type Provider interface {
	GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SteamID) (Core, error)
	EnsurePerson(ctx context.Context, steamID steamid.SteamID) error
}

type DiscordPersonProvider interface {
	GetPersonByDiscordID(ctx context.Context, discordID string) (Core, error)
}

// BaseUser is the smallest profile data set. It's comprised of the most commonly used properties and is
// small enough to be embedded into the JWT to avoid db calls.
type BaseUser interface {
	GetName() string
	GetAvatar() Avatar
	GetSteamID() steamid.SteamID
	HasPermission(privilege permission.Privilege) bool
	GetPrivilege() permission.Privilege
	Path() string
}

type Info interface {
	GetName() string
	GetAvatar() Avatar
	GetSteamID() steamid.SteamID
	GetSteamIDString() string
	GetDiscordID() string
	GetVACBans() int32
	GetGameBans() int32
	GetTimeCreated() time.Time
	Path() string // link.Linkable
	HasPermission(permission permission.Privilege) bool
	Permissions() permission.Privilege
}

// Core is the model used in the webui representing the logged-in user.
type Core struct {
	SteamID         steamid.SteamID      `json:"steam_id"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	Name            string               `json:"name"`
	Avatarhash      string               `json:"avatarhash"`
	DiscordID       string               `json:"discord_id"`
	PatreonID       string               `json:"patreon_id"`
	VacBans         int32                `json:"vac_bans"`
	GameBans        int32                `json:"game_bans"`
	TimeCreated     time.Time            `json:"time_created"`
	BanID           int32                `json:"ban_id"`
}

func (p Core) Permissions() permission.Privilege {
	return p.PermissionLevel
}

func (p Core) HasPermission(privilege permission.Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p Core) GetVACBans() int32 {
	return p.VacBans
}

func (p Core) GetGameBans() int32 {
	return p.GameBans
}

func (p Core) GetTimeCreated() time.Time {
	return p.TimeCreated
}

func (p Core) GetPrivilege() permission.Privilege {
	return p.PermissionLevel
}

func (p Core) GetName() string {
	if p.Name == "" {
		return p.SteamID.String()
	}

	return p.Name
}

func (p Core) GetAvatar() Avatar {
	if p.Avatarhash == "" {
		return "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"
	}

	return Avatar(p.Avatarhash)
}

func (p Core) GetDiscordID() string {
	return p.DiscordID
}

func (p Core) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p Core) GetSteamIDString() string {
	return p.SteamID.String()
}

func (p Core) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}
