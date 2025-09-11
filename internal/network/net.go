package network

import (
	"errors"
	"net"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrNetworkInvalidASNRecord      = errors.New("invalid asn record")
	ErrNetworkInvalidLocationRecord = errors.New("invalid location record")
	ErrNetworkInvalidProxyRecord    = errors.New("invalid proxy record")
)

// type NetworkUsecase interface {
// 	GetASNRecordsByNum(ctx context.Context, asNum int64) ([]NetworkASN, error)
// 	GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (PersonConnections, error)
// 	GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP
// 	QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQuery) ([]PersonConnection, int64, error)
// 	AddConnectionHistory(ctx context.Context, conn *PersonConnection) error
// 	Start(ctx context.Context)
// 	QueryNetwork(ctx context.Context, ip netip.Addr) (NetworkDetails, error)
// 	RefreshLocationData(ctx context.Context) error
// }

// type NetworkRepository interface {
// 	QueryConnections(ctx context.Context, opts ConnectionHistoryQuery) ([]PersonConnection, int64, error)
// 	GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (PersonConnections, error)
// 	AddConnectionHistory(ctx context.Context, conn *PersonConnection) error
// 	GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP
// 	GetASNRecordsByNum(ctx context.Context, asNum int64) ([]NetworkASN, error)
// 	GetASNRecordByIP(ctx context.Context, ipAddr netip.Addr) (NetworkASN, error)
// 	GetLocationRecord(ctx context.Context, ipAddr netip.Addr) (NetworkLocation, error)
// 	GetProxyRecord(ctx context.Context, ipAddr netip.Addr) (NetworkProxy, error)
// 	LoadASN(ctx context.Context, truncate bool, records []ip2location.ASNRecord) error
// 	LoadLocation(ctx context.Context, truncate bool, records []ip2location.LocationRecord) error
// 	LoadProxies(ctx context.Context, truncate bool, records []ip2location.ProxyRecord) error
// }

// PersonIPRecord holds a composite result of the more relevant ip2location results.
type PersonIPRecord struct {
	IPAddr      net.IP
	CreatedOn   time.Time
	CityName    string
	CountryName string
	CountryCode string
	ASName      string
	ASNum       int
	ISP         string
	UsageType   string
	Threat      string
	DomainUsed  string
}

type ConnectionHistoryQuery struct {
	domain.QueryFilter
	domain.SourceIDField
	CIDR    string `json:"cidr,omitempty"`
	ASN     int    `json:"asn,omitempty"`
	Sid64   string `json:"sid64,omitempty"`
	Network string `json:"network,omitempty"`
}

type PersonConnection struct {
	PersonConnectionID int64           `json:"person_connection_id"`
	IPAddr             netip.Addr      `json:"ip_addr"`
	SteamID            steamid.SteamID `json:"steam_id"`
	PersonaName        string          `json:"persona_name"`
	ServerID           int             `json:"server_id"`
	CreatedOn          time.Time       `json:"created_on"`
	ServerNameShort    string          `json:"server_name_short"`
	ServerName         string          `json:"server_name"`
}

type PersonConnections []PersonConnection

type NetworkDetailsQuery struct {
	domain.QueryFilter
	IP netip.Addr `json:"ip"`
}

type NetworkDetails struct {
	Location NetworkLocation `json:"location"`
	Asn      NetworkASN      `json:"asn"`
	Proxy    NetworkProxy    `json:"proxy"`
}

type NetworkLocation struct {
	CIDR        string              `json:"cidr"`
	CountryCode string              `json:"country_code"`
	CountryName string              `json:"country_name"`
	RegionName  string              `json:"region_name"`
	CityName    string              `json:"city_name"`
	LatLong     ip2location.LatLong `json:"lat_long"`
}

type NetworkASN struct {
	CIDR   string `json:"cidr"`
	ASNum  uint64 `json:"as_num"`
	ASName string `json:"as_name"`
}

type NetworkProxy struct {
	CIDR        string                 `json:"cidr"`
	ProxyType   ip2location.ProxyType  `json:"proxy_type"`
	CountryCode string                 `json:"country_code"`
	CountryName string                 `json:"country_name"`
	RegionName  string                 `json:"region_name"`
	CityName    string                 `json:"city_name"`
	ISP         string                 `json:"isp"`
	Domain      string                 `json:"domain"`
	UsageType   ip2location.UsageType  `json:"usage_type"`
	ASN         int64                  `json:"as_num"`  //nolint:tagliatelle
	AS          string                 `json:"as_name"` //nolint:tagliatelle
	LastSeen    time.Time              `json:"last_seen"`
	Threat      ip2location.ThreatType `json:"threat"`
}
