package domain

import "net"

type NetBLocker interface {
	AddWhitelist(id int, network *net.IPNet)
	RemoveWhitelist(id int)
	IsMatch(addr net.IP) (bool, string)
}

type CIDRBlockSource struct {
	CIDRBlockSourceID int    `json:"cidr_block_source_id"`
	Name              string `json:"name"`
	URL               string `json:"url"`
	Enabled           bool   `json:"enabled"`
	TimeStamped
}

type CIDRBlockWhitelist struct {
	CIDRBlockWhitelistID int        `json:"cidr_block_whitelist_id"`
	Address              *net.IPNet `json:"address"`
	TimeStamped
}
