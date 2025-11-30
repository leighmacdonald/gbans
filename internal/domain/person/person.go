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

type Provider interface {
	GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SteamID) (Core, error)
}

type DiscordPersonProvider interface {
	GetPersonByDiscordID(ctx context.Context, discordID string) (Core, error)
}

type Info interface {
	GetName() string
	GetAvatar() Avatar
	GetSteamID() steamid.SteamID
	GetSteamIDString() string
	GetDiscordID() string
	GetVACBans() int
	GetGameBans() int
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
	VacBans         int                  `json:"vac_bans"`
	GameBans        int                  `json:"game_bans"`
	TimeCreated     time.Time            `json:"time_created"`
	BanID           int64                `json:"ban_id"`
}

func (p Core) Permissions() permission.Privilege {
	return p.PermissionLevel
}

func (p Core) HasPermission(privilege permission.Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p Core) GetVACBans() int {
	return p.VacBans
}

func (p Core) GetGameBans() int {
	return p.GameBans
}

func (p Core) GetTimeCreated() time.Time {
	return p.TimeCreated
}

func (p Core) GetName() string {
	if p.Name == "" {
		return p.SteamID.String()
	}

	return p.Name
}

func (p Core) GetAvatar() Avatar {
	if p.Avatarhash == "" {
		return NewAvatar("fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb")
	}

	return NewAvatar(p.Avatarhash)
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
