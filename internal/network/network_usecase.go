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

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type networkUsecase struct {
	repository domain.NetworkRepository
	persons    domain.PersonUsecase
	config     domain.ConfigUsecase
	eb         *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewNetworkUsecase(broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository domain.NetworkRepository, persons domain.PersonUsecase, config domain.ConfigUsecase,
) domain.NetworkUsecase {
	return networkUsecase{
		repository: repository,
		eb:         broadcaster,
		persons:    persons,
		config:     config,
	}
}

func (u networkUsecase) Start(ctx context.Context) {
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

			// Maybe ignore these and wait for connect call to create?
			_, errPerson := u.persons.GetOrCreatePersonBySteamID(ctx, nil, newServerEvent.SID)
			if errPerson != nil && !errors.Is(errPerson, domain.ErrDuplicate) {
				slog.Error("Failed to fetch connecting person", slog.String("steam_id", newServerEvent.SID.String()), log.ErrAttr(errPerson))

				continue
			}

			conn := domain.PersonConnection{
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

func (u networkUsecase) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	return u.repository.AddConnectionHistory(ctx, conn)
}

func (u networkUsecase) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]domain.NetworkASN, error) {
	return u.repository.GetASNRecordsByNum(ctx, asNum)
}

func (u networkUsecase) importDatabase(ctx context.Context, dbName ip2location.DatabaseFile) error {
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

func (u networkUsecase) RefreshLocationData(ctx context.Context) error {
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

func (u networkUsecase) GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (domain.PersonConnections, error) {
	return u.repository.GetPersonIPHistory(ctx, sid64, limit)
}

func (u networkUsecase) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	return u.repository.GetPlayerMostRecentIP(ctx, steamID)
}

func (u networkUsecase) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQuery) ([]domain.PersonConnection, int64, error) {
	if sid, ok := opts.SourceSteamID(ctx); ok {
		opts.Sid64 = sid.Int64()
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
			slog.Error("Received malformed CIDR", log.ErrAttr(errNetwork))

			return nil, 0, domain.ErrInvalidCIDR
		}

		opts.Network = network.String()
	}

	if opts.Sid64 <= 0 || opts.Network == "" {
		return nil, 0, domain.ErrMissingParam
	}

	return u.repository.QueryConnections(ctx, opts)
}

func (u networkUsecase) QueryNetwork(ctx context.Context, address netip.Addr) (domain.NetworkDetails, error) {
	var details domain.NetworkDetails

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
	if errProxy != nil && !errors.Is(errProxy, domain.ErrNoResult) {
		return details, errors.Join(errProxy, domain.ErrNetworkProxyUnknown)
	}

	details.Proxy = proxy

	return details, nil
}
