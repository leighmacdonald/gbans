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
	v1 "github.com/leighmacdonald/gbans/internal/rpc/servers/v1"
	"github.com/leighmacdonald/gbans/internal/rpc/servers/v1/serversv1connect"
	"github.com/maruel/natural"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPC struct {
	serversv1connect.UnimplementedServersServiceHandler

	servers *Servers
}

func NewRPC(servers *Servers) *RPC {
	return &RPC{servers: servers}
}

func (s RPC) State(_ context.Context, req *v1.StateRequest) (*v1.StateResponse, error) {
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
	for _, cs := range servers {
		resp.Servers = append(resp.Servers, &v1.SafeServer{
			ServerId:   ptr.To(cs.ServerID),
			Host:       &cs.Host,
			Port:       ptr.To(uint32(cs.Port)),
			Ip:         &cs.IP,
			Name:       &cs.Name,
			NameShort:  &cs.NameShort,
			Region:     &cs.Region,
			Cc:         &cs.CC,
			Players:    ptr.To(cs.Players),
			MaxPlayers: ptr.To(cs.MaxPlayers),
			Bot:        ptr.To(cs.Bots),
			Map:        &cs.Map,
			GameTypes:  cs.GameTypes,
			Latitude:   ptr.To(float32(cs.Latitude)),
			Longitude:  ptr.To(float32(cs.Longitude)),
			Distance:   ptr.To(float32(cs.Distance)),
			Humans:     ptr.To(cs.Humans),
			Tags:       cs.Tags,
		})
	}

	return &resp, nil
}

func (s RPC) Servers(ctx context.Context, _ *emptypb.Empty) (*v1.ServersResponse, error) {
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

func (s RPC) EditServer(ctx context.Context, req *v1.EditServerRequest) (*v1.EditServerResponse, error) {
	server, errSave := s.servers.Save(ctx, fromRPCServer(req.Server))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Join(errSave, httphelper.ErrInternal))
	}

	return &v1.EditServerResponse{Server: toRPCServer(server)}, nil
}

func (s RPC) DeleteServer(ctx context.Context, req *v1.DeleteServerRequest) (*emptypb.Empty, error) {
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

func fromRPCServer(s *v1.Server) Server {
	return Server{
		ServerID:           ptr.From(s.ServerId),
		ShortName:          ptr.From(s.ShortName),
		Name:               ptr.From(s.Name),
		Address:            ptr.From(s.Address),
		AddressInternal:    ptr.From(s.AddressInternal),
		SDREnabled:         ptr.From(s.SdrEnabled),
		Port:               uint16(ptr.From(s.Port)),
		RCON:               ptr.From(s.Rcon),
		Password:           ptr.From(s.Password),
		IsEnabled:          ptr.From(s.IsEnabled),
		Deleted:            ptr.From(s.Deleted),
		Region:             ptr.From(s.Region),
		CC:                 ptr.From(s.Cc),
		Latitude:           float64(ptr.From(s.Latitude)),
		Longitude:          float64(ptr.From(s.Longitude)),
		LogSecret:          ptr.From(s.LogSecret),
		EnableStats:        ptr.From(s.EnableStats),
		TokenCreatedOn:     s.TokenCreatedOn.AsTime(),
		CreatedOn:          s.CreatedOn.AsTime(),
		UpdatedOn:          s.UpdatedOn.AsTime(),
		DiscordSeedRoleIDs: s.DiscordSeedRoleIds,
		IP:                 net.ParseIP(ptr.From(s.Ip)),
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

func (s RPC) ServersAdmin(ctx context.Context, _ *emptypb.Empty) (*v1.ServersAdminResponse, error) {
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
