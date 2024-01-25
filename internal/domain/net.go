package domain

import (
	"context"
	"net"

	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"go.uber.org/zap"
)

type NetworkUsecase interface {
	GetASNRecordByIP(ctx context.Context, ipAddr net.IP, asnRecord *ip2location.ASNRecord) error
	GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error)
	GetLocationRecord(ctx context.Context, ipAddr net.IP, record *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ipAddr net.IP, proxyRecord *ip2location.ProxyRecord) error
	InsertBlockListData(ctx context.Context, log *zap.Logger, blockListData *ip2location.BlockListData) error
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
