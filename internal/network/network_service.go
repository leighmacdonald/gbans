package network

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strconv"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	v1 "github.com/leighmacdonald/gbans/internal/network/v1"
	"github.com/leighmacdonald/gbans/internal/network/v1/networkv1connect"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// dbUpdateTimeout bounds a full geo database download and import. The RPC
// returns immediately, so the work must not inherit the request's cancellation.
const dbUpdateTimeout = time.Hour

type Service struct {
	// networkv1connect.UnimplementedNetworkServiceHandler

	updateInProgress atomic.Bool
	networks         Networks
}

func NewNetworkService(networks Networks, openGraph *opengraph.Repository, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := networkv1connect.NewNetworkServiceHandler(&Service{networks: networks}, option...)

	openGraph.Register("/admin/network", opengraph.Static("Network History Tools", "Various network querying tools."))
	openGraph.Register("/admin/network/cidrblocks",
		opengraph.Static("CIDR Network Blocks", "Inspect CIDR network blocks."))
	openGraph.Register("/admin/network/ipInfo", opengraph.Static("IP Info", "Look up IP address information."))
	openGraph.Register("/admin/network/playersbyip",
		opengraph.Static("Players By IP", "Find players by IP address."))

	authMiddleware.UserRoute(networkv1connect.NetworkServiceQueryConnectionsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NETWORK_READ))
	authMiddleware.UserRoute(networkv1connect.NetworkServiceQueryNetworkProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NETWORK_READ))
	authMiddleware.UserRoute(networkv1connect.NetworkServiceUpdateDBProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NETWORK_ADMIN))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s *Service) QueryConnections(ctx context.Context, req *v1.QueryConnectionsRequest) (*v1.QueryConnectionsResponse, error) {
	ipHist, errIPHist := s.networks.QueryConnectionHistory(ctx, ConnectionHistoryQuery{
		Filter:      rpc.FromRPC(req.GetFilter()),
		SourceID:    strconv.FormatInt(req.GetSteamId(), 10),
		CIDR:        req.GetCidr(),
		CountryCode: req.GetCountryCode(),
		CountryName: req.GetCountryName(),
		CityName:    req.GetCityName(),
		ServerID:    req.GetServerId(),
	})
	if errIPHist != nil && !errors.Is(errIPHist, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.QueryConnectionsResponse{Connection: make([]*v1.PersonConnection, len(ipHist))}
	for idx, hist := range ipHist {
		resp.Connection[idx] = &v1.PersonConnection{
			PersonConnectionId: &hist.PersonConnectionID,
			IpAddr:             new(hist.IPAddr.String()),
			SteamId:            new(hist.SteamID.Int64()),
			PersonaName:        &hist.PersonaName,
			ServerId:           &hist.ServerID,
		}
	}

	return &resp, nil
}

func (s *Service) QueryNetwork(ctx context.Context, req *v1.QueryNetworkRequest) (*v1.QueryNetworkResponse, error) {
	addr, errAddr := netip.ParseAddr(req.GetIp())
	if errAddr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}
	details, err := s.networks.QueryNetwork(ctx, addr)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.QueryNetworkResponse{Details: &v1.Details{
		Location: toLocation(details.Location),
		Asn:      toASN(details.Asn),
		Proxy:    toProxy(details.Proxy),
	}}

	return &resp, nil
}

func (s *Service) UpdateDB(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if s.updateInProgress.CompareAndSwap(false, true) {
		// The RPC returns immediately, which cancels the request-scoped ctx. The
		// download/import must outlive the response, so detach from that
		// cancellation and apply our own upper bound instead.
		updateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbUpdateTimeout)

		go func() {
			defer cancel()
			defer s.updateInProgress.Store(false)

			if err := s.networks.RefreshLocationData(updateCtx); err != nil {
				slog.Error("Failed to update location data", slog.String("error", err.Error()))
			}
		}()

		return &emptypb.Empty{}, nil
	}

	slog.Warn("Tried to start concurrent location update")

	return &emptypb.Empty{}, nil
}

func toLocation(loc Location) *v1.Location {
	return &v1.Location{
		Cidr:        &loc.CIDR,
		CountryCode: &loc.CountryCode,
		CountryName: &loc.CountryName,
		RegionName:  &loc.RegionName,
		CityName:    &loc.CityName,
		LatLong:     toLatLong(loc.LatLong),
	}
}

func toLatLong(loc ip2location.LatLong) *v1.LatLong {
	return &v1.LatLong{Latitude: &loc.Latitude, Longitude: &loc.Longitude}
}

func toASN(asn ASN) *v1.ASN {
	return &v1.ASN{
		Cidr:   &asn.CIDR,
		AsNum:  &asn.ASNum,
		AsName: &asn.ASName,
	}
}

func toProxy(proxy Proxy) *v1.Proxy {
	return &v1.Proxy{Cidr: &proxy.CIDR}
}
