package discord

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrCommandFailed = errors.New("command failed")
	DmPerms          = false                                  //nolint:gochecknoglobals
	ModPerms         = int64(discordgo.PermissionBanMembers)  //nolint:gochecknoglobals
	UserPerms        = int64(discordgo.PermissionViewChannel) //nolint:gochecknoglobals
)

const (
	OptUserIdentifier   = "user_identifier"
	OptServerIdentifier = "server_identifier"
	OptMessage          = "message"
	OptDuration         = "duration"
	OptASN              = "asn"
	OptIP               = "ip"
	OptMatchID          = "match_id"
	OptBanReason        = "ban_reason"
	OptUnbanReason      = "unban_reason"
	OptBan              = "ban"
	OptSteam            = "steam"
	OptNote             = "note"
	OptCIDR             = "cidr"
	OptPattern          = "pattern"
	OptIsRegex          = "is_regex"
)
