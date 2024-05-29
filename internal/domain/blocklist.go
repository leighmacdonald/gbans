package domain

import (
	"context"
	"encoding/xml"
	"net/netip"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type BlocklistUsecase interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	CreateCIDRBlockSources(ctx context.Context, name string, url string, enabled bool) (CIDRBlockSource, error)
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]WhitelistIP, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *WhitelistIP) error
	CreateCIDRBlockWhitelist(ctx context.Context, address string) (WhitelistIP, error)
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
	UpdateCIDRBlockSource(ctx context.Context, sourceID int, name string, url string, enabled bool) (CIDRBlockSource, error)
	UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (WhitelistIP, error)
	CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (WhitelistSteam, error)
	GetSteamBlockWhitelists(ctx context.Context) ([]WhitelistSteam, error)
	DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error
	Start(ctx context.Context)
}

type BlocklistRepository interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	SaveCIDRBlockSources(ctx context.Context, block *CIDRBlockSource) error
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]WhitelistIP, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *WhitelistIP) error
	SaveCIDRBlockWhitelist(ctx context.Context, whitelist *WhitelistIP) error
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
	TruncateCachedEntries(ctx context.Context) error
	InsertCache(ctx context.Context, list CIDRBlockSource, entries []netip.Prefix) error
	CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (WhitelistSteam, error)
	GetSteamBlockWhitelists(ctx context.Context) ([]WhitelistSteam, error)
	DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error
}

type SteamGroupInfo struct {
	XMLName      xml.Name `xml:"memberList"`
	Text         string   `xml:",chardata"`
	GroupID64    int64    `xml:"groupID64"`
	GroupDetails struct {
		Text          string `xml:",chardata"`
		GroupName     string `xml:"groupName"`
		GroupURL      string `xml:"groupURL"`
		Headline      string `xml:"headline"`
		Summary       string `xml:"summary"`
		AvatarIcon    string `xml:"avatarIcon"`
		AvatarMedium  string `xml:"avatarMedium"`
		AvatarFull    string `xml:"avatarFull"`
		MemberCount   string `xml:"memberCount"`
		MembersInChat string `xml:"membersInChat"`
		MembersInGame string `xml:"membersInGame"`
		MembersOnline string `xml:"membersOnline"`
	} `xml:"groupDetails"`
	MemberCount    string `xml:"memberCount"`
	TotalPages     string `xml:"totalPages"`
	CurrentPage    string `xml:"currentPage"`
	StartingMember string `xml:"startingMember"`
	Members        struct {
		Text      string  `xml:",chardata"`
		SteamID64 []int64 `xml:"steamID64"`
	} `xml:"members"`
}
