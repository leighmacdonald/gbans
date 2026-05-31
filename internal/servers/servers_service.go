package servers

import (
	"context"
	"errors"
	"net"
	"sort"
	"strconv"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	networkv1 "github.com/leighmacdonald/gbans/internal/network/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"github.com/maruel/natural"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	serversv1connect.UnimplementedServersServiceHandler

	servers *Servers
}

func NewServersService(servers *Servers, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := serversv1connect.NewServersServiceHandler(&Service{servers: servers}, option...)

	authMiddleware.UserRoute(serversv1connect.ServersServiceStateProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.UserRoute(serversv1connect.ServersServiceServersProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(serversv1connect.ServersServiceEditServerProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(serversv1connect.ServersServiceDeleteServerProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(serversv1connect.ServersServiceServersAdminProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(serversv1connect.ServersServiceQueryLogsProcedure, rpc.WithMinPermissions(permission.Admin))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func getLatLong(ctx context.Context) (float64, float64) {
	callInfo, ok := connect.CallInfoForHandlerContext(ctx)
	if !ok {
		return 0, 0
	}

	head := callInfo.RequestHeader()
	lat, _ := strconv.ParseFloat(head.Get("CF-IPLatitude"), 64)
	long, _ := strconv.ParseFloat(head.Get("CF-IPLongitude"), 64)

	return lat, long

}

func (s Service) State(ctx context.Context, req *v1.StateRequest) (*v1.StateResponse, error) {
	lat, lon := getLatLong(ctx)
	servers := s.servers.Current()

	for index, srv := range servers {
		servers[index].Distance = float32(distance(float64(srv.Latitude), float64(srv.Longitude), lat, lon))
	}
	sort.Slice(servers, func(i, j int) bool {
		return natural.Less(servers[i].Name, servers[j].Name)
	})

	resp := v1.StateResponse{}
	for _, current := range servers {
		resp.Servers = append(resp.Servers, &v1.SafeServer{
			ServerId:   &current.ServerID,
			Host:       &current.Host,
			Port:       new(uint32(current.Port)),
			Ip:         &current.IP,
			Name:       &current.Name,
			NameShort:  &current.NameShort,
			Region:     &current.Region,
			Cc:         &current.CC,
			Players:    &current.Players,
			MaxPlayers: new(current.MaxPlayerDisplay()),
			Bot:        &current.Bots,
			Map:        &current.Map,
			GameTypes:  current.GameTypes,
			LatLong: &networkv1.LatLong{
				Latitude:  &current.Latitude,
				Longitude: &current.Longitude,
			},
			Distance: &current.Distance,
			Humans:   &current.Humans,
			Tags:     current.Tags,
		})
	}

	return &resp, nil
}

func (s Service) Servers(ctx context.Context, _ *emptypb.Empty) (*v1.ServersResponse, error) {
	fullServers, errServers := s.servers.Servers(ctx, Query{IncludeDisabled: false, IncludeDeleted: false})
	if errServers != nil && !errors.Is(errServers, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errServers, rpc.ErrInternal))
	}

	var resp v1.ServersResponse
	for _, server := range fullServers {
		resp.Servers = append(resp.Servers, &v1.ServerInfoSafe{
			ServerNameLong: &server.Name,
			ServerName:     &server.ShortName,
			ServerId:       new(server.ServerID),
			Colour:         new(""),
		})
	}

	return &resp, nil
}

func (s Service) EditServer(ctx context.Context, req *v1.EditServerRequest) (*v1.EditServerResponse, error) {
	server, errSave := s.servers.Save(ctx, fromRPCServer(req.GetServer()))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditServerResponse{Server: toRPCServer(server)}, nil
}

func (s Service) DeleteServer(ctx context.Context, req *v1.DeleteServerRequest) (*emptypb.Empty, error) {
	serverID := req.GetServerId()
	if serverID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, httphelper.ErrNotFound)
	}

	if err := s.servers.Delete(ctx, serverID); err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, httphelper.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func fromRPCServer(server *v1.Server) Server {
	return Server{
		ServerID:           server.GetServerId(),
		ShortName:          server.GetShortName(),
		Name:               server.GetName(),
		Address:            server.GetAddress(),
		AddressInternal:    server.GetAddressInternal(),
		SDREnabled:         server.GetSdrEnabled(),
		Port:               uint16(server.GetPort()), //nolint:gosec
		RCON:               server.GetRcon(),
		Password:           server.GetPassword(),
		IsEnabled:          server.GetIsEnabled(),
		Deleted:            server.GetDeleted(),
		Region:             server.GetRegion(),
		CC:                 server.GetCc(),
		Latitude:           server.GetLatLong().GetLatitude(),
		Longitude:          server.GetLatLong().GetLongitude(),
		LogSecret:          server.GetLogSecret(),
		EnableStats:        server.GetEnableStats(),
		TokenCreatedOn:     server.GetTokenCreatedOn().AsTime(),
		CreatedOn:          server.GetCreatedOn().AsTime(),
		UpdatedOn:          server.GetUpdatedOn().AsTime(),
		DiscordSeedRoleIDs: server.GetDiscordSeedRoleIds(),
		IP:                 net.ParseIP(server.GetIp()),
	}
}

func toRPCServer(server Server) *v1.Server {
	return &v1.Server{
		ServerId:        &server.ServerID,
		ShortName:       &server.ShortName,
		Name:            &server.Name,
		Address:         &server.Address,
		AddressInternal: &server.AddressInternal,
		SdrEnabled:      &server.SDREnabled,
		Port:            new(uint32(server.Port)),
		Rcon:            &server.RCON,
		Password:        &server.Password,
		IsEnabled:       &server.IsEnabled,
		Deleted:         &server.Deleted,
		Region:          &server.Region,
		Cc:              &server.CC,
		LatLong: &networkv1.LatLong{
			Latitude:  &server.Latitude,
			Longitude: &server.Longitude,
		},
		LogSecret:          &server.LogSecret,
		EnableStats:        &server.EnableStats,
		TokenCreatedOn:     timestamppb.New(server.TokenCreatedOn),
		CreatedOn:          timestamppb.New(server.CreatedOn),
		UpdatedOn:          timestamppb.New(server.UpdatedOn),
		DiscordSeedRoleIds: server.DiscordSeedRoleIDs,
		Ip:                 new(server.IP.String()),
	}
}

func (s Service) ServersAdmin(ctx context.Context, _ *emptypb.Empty) (*v1.ServersAdminResponse, error) {
	fullServers, errServers := s.servers.Servers(ctx, Query{IncludeDisabled: true})
	if errServers != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errServers, rpc.ErrInternal))
	}

	var resp v1.ServersAdminResponse
	for _, server := range fullServers {
		resp.Servers = append(resp.Servers, toRPCServer(server))
	}

	return &resp, nil
}
