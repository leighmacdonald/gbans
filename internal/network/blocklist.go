package network

import (
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
