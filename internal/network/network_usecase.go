package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
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
	nr     domain.NetworkRepository
	pu     domain.PersonUsecase
	config domain.ConfigUsecase
	eb     *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewNetworkUsecase(broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository domain.NetworkRepository, personUsecase domain.PersonUsecase, config domain.ConfigUsecase,
) domain.NetworkUsecase {
	return networkUsecase{
		nr:     repository,
		eb:     broadcaster,
		pu:     personUsecase,
		config: config,
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
			_, errPerson := u.pu.GetOrCreatePersonBySteamID(ctx, newServerEvent.SID)
			if errPerson != nil {
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
			if errChat := u.nr.AddConnectionHistory(lCtx, &conn); errChat != nil {
				slog.Error("Failed to add connection history", log.ErrAttr(errChat))
			}

			cancel()
		}
	}
}

func (u networkUsecase) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	return u.nr.AddConnectionHistory(ctx, conn)
}

func (u networkUsecase) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]domain.NetworkASN, error) {
	return u.nr.GetASNRecordsByNum(ctx, asNum)
}

func (u networkUsecase) importDatabase(ctx context.Context, dbName ip2location.DatabaseFile) error {
	conf := u.config.Config()

	linesInput := make(chan any, 1000)

	go func() {
		defer close(linesInput)

		errRead := ip2location.LineReader(conf.GeoLocation.CachePath, dbName, linesInput)
		if errRead != nil {
			slog.Error("Failed to read data", log.ErrAttr(errRead))
		}

		slog.Debug("Line reader completed")
	}()

	var rows []any

	first := true

	for {
		select {
		case line, ok := <-linesInput:
			if ok {
				rows = append(rows, line)
			}

			if !ok || len(rows) == 1000 {
				var err error

				switch dbName {
				case ip2location.GeoDatabaseLocationFile4:
					err = u.nr.LoadLocation(ctx, first, rows)
				case ip2location.GeoDatabaseASNFile4:
					err = u.nr.LoadASN(ctx, first, rows)
				case ip2location.GeoDatabaseProxyFile:
					err = u.nr.LoadProxies(ctx, first, rows)
				}

				if err != nil {
					return err
				}

				clear(rows)

				first = false
			}
		case <-ctx.Done():
			return nil
		}
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
	return u.nr.GetPersonIPHistory(ctx, sid64, limit)
}

func (u networkUsecase) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	return u.nr.GetPlayerMostRecentIP(ctx, steamID)
}

func (u networkUsecase) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQuery) ([]domain.PersonConnection, int64, error) {
	if sid, ok := opts.SourceSteamID(ctx); ok {
		opts.Sid64 = sid.Int64()
	}

	if opts.QueryFilter.Limit > 1000 {
		opts.QueryFilter.Limit = 1000
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

	if !(opts.Sid64 > 0 || opts.Network != "") {
		return nil, 0, domain.ErrMissingParam
	}

	return u.nr.QueryConnections(ctx, opts)
}

func (u networkUsecase) QueryNetwork(ctx context.Context, address netip.Addr) (domain.NetworkDetails, error) {
	var details domain.NetworkDetails

	if !address.IsValid() {
		return details, domain.ErrNetworkInvalidIP
	}

	location, errLocation := u.nr.GetLocationRecord(ctx, address)
	if errLocation != nil {
		return details, errors.Join(errLocation, domain.ErrNetworkLocationUnknown)
	}

	details.Location = location

	asn, errASN := u.nr.GetASNRecordByIP(ctx, address)
	if errASN != nil {
		return details, errors.Join(errASN, domain.ErrNetworkASNUnknown)
	}

	details.Asn = asn

	proxy, errProxy := u.nr.GetProxyRecord(ctx, address)
	if errProxy != nil && !errors.Is(errProxy, domain.ErrNoResult) {
		return details, errors.Join(errProxy, domain.ErrNetworkProxyUnknown)
	}

	details.Proxy = proxy

	return details, nil
}
