package blocklist

import (
	"encoding/xml"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type CIDRBlockSource struct {
	CIDRBlockSourceID int       `json:"cidr_block_source_id"`
	Name              string    `json:"name"`
	URL               string    `json:"url"`
	Enabled           bool      `json:"enabled"`
	CreatedOn         time.Time `json:"created_on"`
	UpdatedOn         time.Time `json:"updated_on"`
}

type WhitelistIP struct {
	CIDRBlockWhitelistID int        `json:"cidr_block_whitelist_id"`
	Address              *net.IPNet `json:"address"`
	CreatedOn            time.Time  `json:"created_on"`
	UpdatedOn            time.Time  `json:"updated_on"`
}

type WhitelistSteam struct {
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	domain.SteamIDField
	Personaname string `json:"personaname"`
	AvatarHash  string `json:"avatar_hash"`
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
