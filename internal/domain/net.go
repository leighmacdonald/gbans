package domain

import (
	"context"
	"net"

	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type NetworkUsecase interface {
	LoadNetBlocks(ctx context.Context) error
	GetASNRecordByIP(ctx context.Context, ipAddr net.IP, asnRecord *ip2location.ASNRecord) error
	GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error)
	GetLocationRecord(ctx context.Context, ipAddr net.IP, record *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ipAddr net.IP, proxyRecord *ip2location.ProxyRecord) error
	InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error
	GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (PersonConnections, error)
	GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SID64) net.IP
	QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQueryFilter) ([]PersonConnection, int64, error)
	AddConnectionHistory(ctx context.Context, conn *PersonConnection) error
	IsMatch(addr net.IP) (string, bool)
	AddWhitelist(id int, network *net.IPNet)
	RemoveWhitelist(id int)
	AddRemoteSource(ctx context.Context, name string, url string) (int64, error)
}
type NetworkRepository interface {
	QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQueryFilter) ([]PersonConnection, int64, error)
	GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (PersonConnections, error)
	AddConnectionHistory(ctx context.Context, conn *PersonConnection) error
	GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SID64) net.IP
	GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error)
	GetASNRecordByIP(ctx context.Context, ipAddr net.IP, asnRecord *ip2location.ASNRecord) error
	GetLocationRecord(ctx context.Context, ipAddr net.IP, record *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ipAddr net.IP, proxyRecord *ip2location.ProxyRecord) error
	InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error
	GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error)
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
