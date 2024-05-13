package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type networkUsecase struct {
	nr      domain.NetworkRepository
	bl      domain.BlocklistUsecase
	pu      domain.PersonUsecase
	blocker *Blocker
	eb      *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewNetworkUsecase(broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	repository domain.NetworkRepository, blocklistUsecase domain.BlocklistUsecase, personUsecase domain.PersonUsecase,
) domain.NetworkUsecase {
	return networkUsecase{
		nr:      repository,
		bl:      blocklistUsecase,
		blocker: NewBlocker(),
		eb:      broadcaster,
		pu:      personUsecase,
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
				slog.Error("Failed to load Person", log.ErrAttr(errPerson))

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

func (u networkUsecase) LoadNetBlocks(ctx context.Context) error {
	sources, errSource := u.bl.GetCIDRBlockSources(ctx)
	if errSource != nil {
		return errors.Join(errSource, domain.ErrInitNetBlocks)
	}

	var total atomic.Int64

	waitGroup := sync.WaitGroup{}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		waitGroup.Add(1)

		go func(src domain.CIDRBlockSource) {
			defer waitGroup.Done()

			count, errAdd := u.blocker.AddRemoteSource(ctx, src.Name, src.URL)
			if errAdd != nil {
				slog.Error("Could not load remote source URL",
					slog.String("name", src.Name), slog.String("url", src.URL))
			}

			total.Add(count)
		}(source)
	}

	waitGroup.Wait()

	whitelists, errWhitelists := u.bl.GetCIDRBlockWhitelists(ctx)
	if errWhitelists != nil {
		if !errors.Is(errWhitelists, domain.ErrNoResult) {
			return errors.Join(errWhitelists, domain.ErrInitNetWhitelist)
		}
	}

	for _, whitelist := range whitelists {
		u.blocker.AddWhitelist(whitelist.CIDRBlockWhitelistID, whitelist.Address)
	}

	slog.Info("Loaded cidr block lists",
		slog.Int64("cidr_blocks", total.Load()), slog.Int("whitelisted", len(whitelists)))

	return nil
}

func (u networkUsecase) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	return u.nr.AddConnectionHistory(ctx, conn)
}

func (u networkUsecase) IsMatch(addr netip.Addr) (string, bool) {
	return u.blocker.IsMatch(addr)
}

func (u networkUsecase) AddWhitelist(id int, network *net.IPNet) {
	u.blocker.AddWhitelist(id, network)
}

func (u networkUsecase) RemoveWhitelist(id int) {
	u.blocker.RemoveWhitelist(id)
}

func (u networkUsecase) AddRemoteSource(ctx context.Context, name string, url string) (int64, error) {
	return u.blocker.AddRemoteSource(ctx, name, url)
}

func (u networkUsecase) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]domain.NetworkASN, error) {
	return u.nr.GetASNRecordsByNum(ctx, asNum)
}

func (u networkUsecase) InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error {
	return u.nr.InsertBlockListData(ctx, blockListData)
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
