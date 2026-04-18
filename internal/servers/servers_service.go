package servers

import (
	"context"
	"errors"
	"net"
	"sort"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/ptr"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"github.com/maruel/natural"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ServersService struct {
	serversv1connect.UnimplementedServersServiceHandler

	servers *Servers
}

func NewServersService(servers *Servers) *ServersService {
	return &ServersService{servers: servers}
}

func (s ServersService) State(_ context.Context, req *v1.StateRequest) (*v1.StateResponse, error) {
	var (
		// TODO
		lat = float64(ptr.From(req.LatLong.Latitude))
		lon = float64(ptr.From(req.LatLong.Longitude))
		// region := ctx.GetHeader("cf-region-code")
		servers = s.servers.Current()
	)

	for index, srv := range servers {
		servers[index].Distance = distance(srv.Latitude, srv.Longitude, lat, lon)
	}
	sort.Slice(servers, func(i, j int) bool {
		return natural.Less(servers[i].Name, servers[j].Name)
	})

	resp := v1.StateResponse{}
	for _, current := range servers {
		resp.Servers = append(resp.Servers, &v1.SafeServer{
			ServerId:   ptr.To(current.ServerID),
			Host:       &current.Host,
			Port:       ptr.To(uint32(current.Port)),
			Ip:         &current.IP,
			Name:       &current.Name,
			NameShort:  &current.NameShort,
			Region:     &current.Region,
			Cc:         &current.CC,
			Players:    ptr.To(current.Players),
			MaxPlayers: ptr.To(current.MaxPlayers),
			Bot:        ptr.To(current.Bots),
			Map:        &current.Map,
			GameTypes:  current.GameTypes,
			Latitude:   ptr.To(float32(current.Latitude)),
			Longitude:  ptr.To(float32(current.Longitude)),
			Distance:   ptr.To(float32(current.Distance)),
			Humans:     ptr.To(current.Humans),
			Tags:       current.Tags,
		})
	}

	return &resp, nil
}

func (s ServersService) Servers(ctx context.Context, _ *emptypb.Empty) (*v1.ServersResponse, error) {
	fullServers, errServers := s.servers.Servers(ctx, Query{IncludeDisabled: false, IncludeDeleted: false})
	if errServers != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errServers, httphelper.ErrInternal))
	}

	var resp v1.ServersResponse
	for _, server := range fullServers {
		resp.Servers = append(resp.Servers, &v1.ServerInfoSafe{
			ServerNameLong: &server.Name,
			ServerName:     &server.ShortName,
			ServerId:       ptr.To(server.ServerID),
			Colour:         ptr.To(""),
		})
	}

	return &resp, nil
}

func (s ServersService) EditServer(ctx context.Context, req *v1.EditServerRequest) (*v1.EditServerResponse, error) {
	server, errSave := s.servers.Save(ctx, fromRPCServer(req.Server))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errSave, httphelper.ErrInternal))
	}

	return &v1.EditServerResponse{Server: toRPCServer(server)}, nil
}

func (s ServersService) DeleteServer(ctx context.Context, req *v1.DeleteServerRequest) (*emptypb.Empty, error) {
	if req.ServerId == nil || *req.ServerId <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, httphelper.ErrNotFound)
	}

	if err := s.servers.Delete(ctx, *req.ServerId); err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, httphelper.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, httphelper.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func fromRPCServer(server *v1.Server) Server {
	return Server{
		ServerID:           ptr.From(server.ServerId),
		ShortName:          ptr.From(server.ShortName),
		Name:               ptr.From(server.Name),
		Address:            ptr.From(server.Address),
		AddressInternal:    ptr.From(server.AddressInternal),
		SDREnabled:         ptr.From(server.SdrEnabled),
		Port:               uint16(ptr.From(server.Port)),
		RCON:               ptr.From(server.Rcon),
		Password:           ptr.From(server.Password),
		IsEnabled:          ptr.From(server.IsEnabled),
		Deleted:            ptr.From(server.Deleted),
		Region:             ptr.From(server.Region),
		CC:                 ptr.From(server.Cc),
		Latitude:           float64(ptr.From(server.Latitude)),
		Longitude:          float64(ptr.From(server.Longitude)),
		LogSecret:          ptr.From(server.LogSecret),
		EnableStats:        ptr.From(server.EnableStats),
		TokenCreatedOn:     server.TokenCreatedOn.AsTime(),
		CreatedOn:          server.CreatedOn.AsTime(),
		UpdatedOn:          server.UpdatedOn.AsTime(),
		DiscordSeedRoleIDs: server.DiscordSeedRoleIds,
		IP:                 net.ParseIP(ptr.From(server.Ip)),
	}
}

func toRPCServer(server Server) *v1.Server {
	return &v1.Server{
		ServerId:           &server.ServerID,
		ShortName:          &server.ShortName,
		Name:               &server.Name,
		Address:            &server.Address,
		AddressInternal:    &server.AddressInternal,
		SdrEnabled:         &server.SDREnabled,
		Port:               ptr.To(int32(server.Port)),
		Rcon:               &server.RCON,
		Password:           &server.Password,
		IsEnabled:          &server.IsEnabled,
		Deleted:            &server.Deleted,
		Region:             &server.Region,
		Cc:                 &server.CC,
		Latitude:           ptr.To(float32(server.Latitude)),
		Longitude:          ptr.To(float32(server.Longitude)),
		LogSecret:          &server.LogSecret,
		EnableStats:        &server.EnableStats,
		TokenCreatedOn:     timestamppb.New(server.TokenCreatedOn),
		CreatedOn:          timestamppb.New(server.CreatedOn),
		UpdatedOn:          timestamppb.New(server.UpdatedOn),
		DiscordSeedRoleIds: server.DiscordSeedRoleIDs,
		Ip:                 ptr.To(server.IP.String()),
	}
}

func (s ServersService) ServersAdmin(ctx context.Context, _ *emptypb.Empty) (*v1.ServersAdminResponse, error) {
	fullServers, errServers := s.servers.Servers(ctx, Query{IncludeDisabled: true})
	if errServers != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errServers, httphelper.ErrInternal))
	}

	var resp v1.ServersAdminResponse
	for _, server := range fullServers {
		resp.Servers = append(resp.Servers, toRPCServer(server))
	}

	return &resp, nil
}
