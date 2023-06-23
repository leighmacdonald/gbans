// Package model defines common model structures used in many places throughout the application.
package model

import (
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type BDIds struct {
	FileInfo struct {
		Authors     []string `json:"authors"`
		Description string   `json:"description"`
		Title       string   `json:"title"`
		UpdateURL   string   `json:"update_url"`
	} `json:"file_info"`
	Schema  string `json:"$schema"`
	Players []struct {
		Steamid    int64    `json:"steamid"`
		Attributes []string `json:"attributes"`
		LastSeen   struct {
			PlayerName string `json:"player_name"`
			Time       int    `json:"time"`
		} `json:"last_seen"`
	} `json:"players"`
	Version int `json:"version"`
}

type SimplePerson struct {
	SteamID     steamid.SID64 `json:"steam_id"`
	PersonaName string        `json:"persona_name"`
	Avatar      string        `json:"avatar"`
	AvatarFull  string        `json:"avatar_full"`
}

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID         steamid.SID64    `db:"steam_id" json:"steam_id,string"`
	CreatedOn       time.Time        `json:"created_on"`
	UpdatedOn       time.Time        `json:"updated_on"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	DiscordID       string           `json:"discord_id"`
	Name            string           `json:"name"`
	Avatar          string           `json:"avatar"`
	AvatarFull      string           `json:"avatarfull"`
	BanID           int64            `json:"ban_id"`
	Muted           bool             `json:"muted"`
}

func (p UserProfile) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// NewUserProfile allocates a new default person instance.
func NewUserProfile(sid64 steamid.SID64) UserProfile {
	t0 := config.Now()

	return UserProfile{
		SteamID:         sid64,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PermissionLevel: consts.PUser,
		Name:            "Guest",
	}
}

// ServerEvent is a flat struct encapsulating a parsed log event.
type ServerEvent struct {
	Server store.Server
	*logparse.Results
}

type LogFilePayload struct {
	Server store.Server
	Lines  []string
	Map    string
}
