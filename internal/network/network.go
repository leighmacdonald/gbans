package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrNetworkInvalidIP       = errors.New("invalid ip/cidr")
	ErrNetworkLocationUnknown = errors.New("unknown location record")
	ErrNetworkASNUnknown      = errors.New("unknown asn record")
	ErrNetworkProxyUnknown    = errors.New("no proxy record")
	ErrUpdateInProgress       = errors.New("location database update already in progress")
)

type Config struct {
	sync.RWMutex

	SDREnabled bool
}

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
	httphelper.SourceIDField

	CIDR        string
	ASNum       int32
	ASName      string
	CountryCode string
	CountryName string
	CityName    string
	ServerID    []int32
}

type PersonConnection struct {
	PersonConnectionID int64
	IPAddr             netip.Addr
	SteamID            steamid.SteamID
	PersonaName        string
	ServerID           int32
	CreatedOn          time.Time
	ServerNameShort    string
	ServerName         string
	ASNum              int32
	ASName             string
	CountryCode        string
	CountryName        string
	CityName           string
	LatLong            ip2location.LatLong
}

type PersonConnections []PersonConnection

type DetailsQuery struct {
	query.Filter

	IP netip.Addr
}

type Details struct {
	Location Location
	Asn      ASN
	Proxy    Proxy
}

type Location struct {
	CIDR        string
	CountryCode string
	CountryName string
	RegionName  string
	CityName    string
	LatLong     ip2location.LatLong
}

type ASN struct {
	CIDR   string
	ASNum  uint64
	ASName string
}

type Proxy struct {
	CIDR        string
	ProxyType   ip2location.ProxyType
	CountryCode string
	CountryName string
	RegionName  string
	CityName    string
	ISP         string
	Domain      string
	UsageType   ip2location.UsageType
	ASN         int64  //nolint:tagliatelle
	AS          string //nolint:tagliatelle
	LastSeen    time.Time
	Threat      ip2location.ThreatType
}

type Networks struct {
	*Config

	geoConf    *ip2location.Config
	repository Repository
	eb         *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
	// updating guards against concurrent location database refreshes from the
	// auto updater, the admin ui and the cli.
	updating *atomic.Bool
}

func NewNetworks(broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository Repository, config *Config, geoConf *ip2location.Config,
) Networks {
	return Networks{
		Config:     config,
		repository: repository,
		eb:         broadcaster,
		geoConf:    geoConf,
		updating:   &atomic.Bool{},
	}
}

func (u Networks) Start(ctx context.Context) {
	serverEventChan := make(chan logparse.ServerEvent)
	if errRegister := u.eb.Consume(serverEventChan, logparse.Connected); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", slog.String("error", errRegister.Error()))

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
				slog.Error("Failed to add connection history", slog.String("error", errChat.Error()))
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
	filePath := path.Join(u.geoConf.CachePath, string(dbName))

	switch dbName {
	case ip2location.GeoDatabaseLocationFile4:
		return ip2location.ReadLocationRecords(ctx, filePath, false, u.repository.LoadLocation)
	case ip2location.GeoDatabaseASNFile4:
		return ip2location.ReadASNRecords(ctx, filePath, false, u.repository.LoadASN)
	case ip2location.GeoDatabaseProxyFile:
		return ip2location.ReadProxyRecords(ctx, filePath, u.repository.LoadProxies)
	default:
		return ErrNetworkLocationUnknown
	}
}

func (u Networks) RefreshLocationData(ctx context.Context) error {
	if u.updating != nil {
		if !u.updating.CompareAndSwap(false, true) {
			return ErrUpdateInProgress
		}
		defer u.updating.Store(false)
	}

	if errUpdate := ip2location.Update(ctx, u.geoConf.CachePath, u.geoConf.Token); errUpdate != nil {
		return errUpdate
	}

	for _, dbName := range ip2location.ImportedFiles {
		if err := u.importDatabase(ctx, dbName); err != nil {
			return err
		}
	}

	return nil
}

// StartGeoUpdater re-downloads and imports the ip2location databases whenever
// the cached files are older than the configured update interval, so the geo
// data stays current without relying on an external cron job. It checks once
// on start (which also populates a fresh install) and then on an interval
// ticker. A non-positive interval, missing token or disabled geolocation
// leaves the function as a no-op, keeping `gbans refresh location` the manual
// escape hatch.
func (u Networks) StartGeoUpdater(ctx context.Context) {
	if u.geoConf == nil || !u.geoConf.Enabled || u.geoConf.Token == "" || u.geoConf.UpdateIntervalDays <= 0 {
		return
	}

	interval := time.Duration(u.geoConf.UpdateIntervalDays) * 24 * time.Hour
	slog.Info("Starting geo location auto updater", slog.Duration("interval", interval))

	run := func() {
		if !ip2location.Stale(u.geoConf.CachePath, interval, ip2location.ImportedFiles...) {
			slog.Debug("Geo location database is up to date")

			return
		}

		if err := u.RefreshLocationData(ctx); err != nil {
			slog.Error("Failed to auto update geo location database", slog.String("error", err.Error()))

			// A failed run can leave freshly downloaded files behind that were
			// never imported. Drop them so the next cycle treats the cache as
			// stale again and retries the download and import.
			if !errors.Is(err, ErrUpdateInProgress) {
				for _, file := range ip2location.ImportedFiles {
					if errRemove := os.Remove(filepath.Join(u.geoConf.CachePath, string(file))); errRemove != nil {
						slog.Warn("Failed to remove stale geo database file",
							slog.String("file", string(file)), slog.String("error", errRemove.Error()))
					}
				}
			}

			return
		}

		slog.Info("Geo location database auto update complete")
	}

	run()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (u Networks) GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (PersonConnections, error) {
	return u.repository.GetPersonIPHistory(ctx, sid64, limit)
}

func (u Networks) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	return u.repository.GetPlayerMostRecentIP(ctx, steamID)
}

func (u Networks) QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQuery) ([]PersonConnection, error) {
	if sid, ok := opts.SourceSteamID(ctx); ok {
		opts.SourceID = sid.String()
	}

	if opts.Limit > 1000 {
		opts.Limit = 1000
	}

	if opts.CIDR != "" {
		if !strings.Contains(opts.CIDR, "/") {
			opts.CIDR += "/32"
		}

		_, network, errNetwork := net.ParseCIDR(opts.CIDR)
		if errNetwork != nil {
			slog.Error("Received malformed CIDR", slog.String("error", errNetwork.Error()))

			return nil, ErrNetworkInvalidIP
		}

		opts.CIDR = network.String()
	}

	return u.repository.QueryConnections(ctx, opts)
}

// LocationByIP resolves a public IP to coordinates using the imported geoip
// database. Private, loopback and unresolvable addresses yield an error.
func (u Networks) LocationByIP(ctx context.Context, address netip.Addr) (ip2location.LatLong, error) {
	if !address.IsValid() || !address.IsGlobalUnicast() {
		return ip2location.LatLong{}, ErrNetworkInvalidIP
	}

	record, err := u.repository.GetLocationRecord(ctx, address)
	if err != nil {
		return ip2location.LatLong{}, err
	}

	return record.LatLong, nil
}

func (u Networks) QueryNetwork(ctx context.Context, address netip.Addr) (Details, error) {
	var details Details
	if !address.IsValid() {
		return details, ErrNetworkInvalidIP
	}

	location, _ := u.repository.GetLocationRecord(ctx, address)
	if location.CIDR != "" {
		details.Location = location
	}

	asn, _ := u.repository.GetASNRecordByIP(ctx, address)
	if asn.ASNum > 0 {
		details.Asn = asn
	}

	proxy, errProxy := u.repository.GetProxyRecord(ctx, address)
	if errProxy != nil && !errors.Is(errProxy, database.ErrNoResult) {
		return details, errors.Join(errProxy, ErrNetworkProxyUnknown)
	}

	details.Proxy = proxy

	return details, nil
}
