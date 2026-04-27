package network

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	v1 "github.com/leighmacdonald/gbans/internal/network/v1"
	"github.com/leighmacdonald/gbans/internal/network/v1/networkv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NetworkService struct {
	networkv1connect.UnimplementedNetworkServiceHandler

	networks Networks
}

func NewNetworkService(networks Networks, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := networkv1connect.NewNetworkServiceHandler(NetworkService{networks: networks}, option...)

	authMiddleware.AuthedRoute(networkv1connect.NetworkServiceQueryConnectionsProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(networkv1connect.NetworkServiceQueryNetworkProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(networkv1connect.NetworkServiceUpdateDBProcedure, rpc.WithMinPermissions(permission.Admin))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s NetworkService) QueryConnections(ctx context.Context, req *v1.QueryConnectionsRequest) (*v1.QueryConnectionsResponse, error) {
	ipHist, errIPHist := s.networks.QueryConnectionHistory(ctx, ConnectionHistoryQuery{
		Filter:        rpc.FromRPC(req.GetFilter()),
		SourceIDField: httphelper.SourceIDField{},
		CIDR:          req.GetCidr(),
		CountryCode:   req.GetCountryCode(),
		CountryName:   req.GetCountryName(),
		CityName:      req.GetCityName(),
		ServerID:      req.GetServerId(),
	})
	if errIPHist != nil && !errors.Is(errIPHist, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.QueryConnectionsResponse{Connection: make([]*v1.PersonConnection, len(ipHist))}
	for idx, hist := range ipHist {
		resp.Connection[idx] = &v1.PersonConnection{
			PersonConnectionId: &hist.PersonConnectionID,
			IpAddr:             ptr.To(hist.IPAddr.String()),
			SteamId:            ptr.To(hist.SteamID.Int64()),
			PersonaName:        &hist.PersonaName,
			ServerId:           &hist.ServerID,
		}
	}

	return &resp, nil
}

func (s NetworkService) QueryNetwork(ctx context.Context, req *v1.QueryNetworkRequest) (*v1.QueryNetworkResponse, error) {
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

var updateInProgress atomic.Bool

func (s NetworkService) UpdateDB(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if !updateInProgress.Load() {
		go func() {
			updateInProgress.Store(true)

			if err := s.networks.RefreshLocationData(ctx); err != nil {
				slog.Error("Failed to update location data", slog.String("error", err.Error()))
			}

			updateInProgress.Store(false)
		}()

		return &emptypb.Empty{}, nil
	}

	slog.Warn("Tried to start concurrent location update")

	return &emptypb.Empty{}, nil
}
