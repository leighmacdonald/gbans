package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"path"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrNetworkInvalidASNRecord      = errors.New("invalid asn record")
	ErrNetworkInvalidLocationRecord = errors.New("invalid location record")
	ErrNetworkInvalidProxyRecord    = errors.New("invalid proxy record")
)

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
	query.Filter
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

type DetailsQuery struct {
	query.Filter
	IP netip.Addr `json:"ip"`
}

type Details struct {
	Location Location `json:"location"`
	Asn      ASN      `json:"asn"`
	Proxy    Proxy    `json:"proxy"`
}

type Location struct {
	CIDR        string              `json:"cidr"`
	CountryCode string              `json:"country_code"`
	CountryName string              `json:"country_name"`
	RegionName  string              `json:"region_name"`
	CityName    string              `json:"city_name"`
	LatLong     ip2location.LatLong `json:"lat_long"`
}

type ASN struct {
	CIDR   string `json:"cidr"`
	ASNum  uint64 `json:"as_num"`
	ASName string `json:"as_name"`
}

type Proxy struct {
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

type Networks struct {
	repository Repository
	config     *config.Configuration
	eb         *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewNetworks(broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository Repository, config *config.Configuration,
) Networks {
	return Networks{
		repository: repository,
		eb:         broadcaster,
		config:     config,
	}
}

func (u Networks) Start(ctx context.Context) {
	serverEventChan := make(chan logparse.ServerEvent)
	if errRegister := u.eb.Consume(serverEventChan, logparse.Connected); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", log.ErrAttr(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			newServerEvent, ok := evt.Event.(logparse.ConnectedEvt)
			if !ok {
				continue
			}

			if newServerEvent.Address == "" {
				slog.Warn("Empty Person message body, skipping")

				continue
			}

			parsedAddr, errParsedAddr := netip.ParseAddr(newServerEvent.Address)
			if errParsedAddr != nil {
				slog.Warn("Received invalid address", slog.String("addr", newServerEvent.Address))

				continue
			}

			conn := PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     newServerEvent.SID,
				PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
				CreatedOn:   newServerEvent.CreatedOn,
				ServerID:    evt.ServerID,
			}

			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := u.repository.AddConnectionHistory(lCtx, &conn); errChat != nil {
				slog.Error("Failed to add connection history", log.ErrAttr(errChat))
			}

			cancel()
		}
	}
}

func (u Networks) AddConnectionHistory(ctx context.Context, conn *PersonConnection) error {
	return u.repository.AddConnectionHistory(ctx, conn)
}

func (u Networks) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]ASN, error) {
	return u.repository.GetASNRecordsByNum(ctx, asNum)
}

func (u Networks) importDatabase(ctx context.Context, dbName ip2location.DatabaseFile) error {
	conf := u.config.Config()
	filePath := path.Join(conf.GeoLocation.CachePath, string(dbName))

	switch dbName {
	case ip2location.GeoDatabaseLocationFile4:
		return ip2location.ReadLocationRecords(ctx, filePath, false, u.repository.LoadLocation)
	case ip2location.GeoDatabaseASNFile4:
		return ip2location.ReadASNRecords(ctx, filePath, false, u.repository.LoadASN)
	case ip2location.GeoDatabaseProxyFile:
		return ip2location.ReadProxyRecords(ctx, filePath, u.repository.LoadProxies)
	default:
		return domain.ErrNetworkLocationUnknown
	}
}

func (u Networks) RefreshLocationData(ctx context.Context) error {
	conf := u.config.Config()

	if errUpdate := ip2location.Update(ctx, conf.GeoLocation.CachePath, conf.GeoLocation.Token); errUpdate != nil {
		return errUpdate
	}

	for _, dbName := range []ip2location.DatabaseFile{ip2location.GeoDatabaseLocationFile4, ip2location.GeoDatabaseASNFile4, ip2location.GeoDatabaseProxyFile} {
		if err := u.importDatabase(ctx, dbName); err != nil {
			return err
		}
	}

	return nil
}

func (u Networks) GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (PersonConnections, error) {
	return u.repository.GetPersonIPHistory(ctx, sid64, limit)
}

func (u Networks) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	return u.repository.GetPlayerMostRecentIP(ctx, steamID)
}

func (u Networks) QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQuery) ([]PersonConnection, int64, error) {
	if sid, ok := opts.SourceSteamID(ctx); ok {
		opts.Sid64 = sid.String()
	}

	if opts.Limit > 1000 {
		opts.Limit = 1000
	}

	if opts.CIDR != "" {
		if !strings.Contains(opts.CIDR, "/") {
			opts.CIDR += maskSingleHost
		}

		_, network, errNetwork := net.ParseCIDR(opts.CIDR)
		if errNetwork != nil {
			slog.Error("Received malformed CIDR", log.ErrAttr(errNetwork))

			return nil, 0, ErrInvalidCIDR
		}

		opts.Network = network.String()
	}

	if opts.Sid64 == "" && opts.Network == "" {
		return nil, 0, domain.ErrMissingParam
	}

	return u.repository.QueryConnections(ctx, opts)
}

func (u Networks) QueryNetwork(ctx context.Context, address netip.Addr) (Details, error) {
	var details Details

	if !address.IsValid() {
		return details, domain.ErrNetworkInvalidIP
	}

	location, errLocation := u.repository.GetLocationRecord(ctx, address)
	if errLocation != nil {
		return details, errors.Join(errLocation, domain.ErrNetworkLocationUnknown)
	}

	details.Location = location

	asn, errASN := u.repository.GetASNRecordByIP(ctx, address)
	if errASN != nil {
		return details, errors.Join(errASN, domain.ErrNetworkASNUnknown)
	}

	details.Asn = asn

	proxy, errProxy := u.repository.GetProxyRecord(ctx, address)
	if errProxy != nil && !errors.Is(errProxy, database.ErrNoResult) {
		return details, errors.Join(errProxy, domain.ErrNetworkProxyUnknown)
	}

	details.Proxy = proxy

	return details, nil
}
