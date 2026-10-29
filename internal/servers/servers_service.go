package servers

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network"
	networkv1 "github.com/leighmacdonald/gbans/internal/network/v1"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"github.com/maruel/natural"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	serversv1connect.UnimplementedServersServiceHandler

	servers   *Servers
	locations *network.Networks
}

func NewServersService(servers *Servers, locations *network.Networks, openGraph *opengraph.Repository, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := serversv1connect.NewServersServiceHandler(&Service{servers: servers, locations: locations}, option...)

	openGraph.Register("/admin/servers", opengraph.Static("Edit Servers", "Manage game servers."))
	openGraph.Register("/admin/serverlogs", opengraph.Static("Server Logs", "Browse server logs."))
	openGraph.Register("/servers", opengraph.Static("Servers", "Browse and connect to active game servers."))

	authMiddleware.PublicRoute(serversv1connect.ServersServiceStateProcedure)
	authMiddleware.UserRoute(serversv1connect.ServersServiceEditServerProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_WRITE))
	authMiddleware.UserRoute(serversv1connect.ServersServiceDeleteServerProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_WRITE))
	authMiddleware.UserRoute(serversv1connect.ServersServiceServersAdminProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_READ))
	authMiddleware.UserRoute(serversv1connect.ServersServiceQueryLogsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_READ))
	authMiddleware.UserRoute(serversv1connect.ServersServiceStreamLogsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_READ))
	authMiddleware.UserRoute(serversv1connect.ServersServiceExecCommandProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_WRITE))
	authMiddleware.UserRoute(serversv1connect.ServersServiceClearLogsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_SERVER_WRITE))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func validLatLong(lat float64, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && !(lat == 0 && lon == 0)
}

// clientLatLong resolves the caller's coordinates for distance sorting and
// map centering. It prefers Cloudflare's injected CF-IPLatitude /
// CF-IPLongitude headers and falls back to a geoip database lookup of the
// client IP (CF-Connecting-IP, X-Forwarded-For, X-Real-IP). Returns zeros
// when no location can be determined.
func (s Service) clientLatLong(ctx context.Context) (float64, float64) {
	callInfo, found := connect.CallInfoForHandlerContext(ctx)
	if !found {
		return 0, 0
	}

	head := callInfo.RequestHeader()
	if lat, errLat := strconv.ParseFloat(head.Get("Cf-Iplatitude"), 64); errLat == nil {
		if lon, errLon := strconv.ParseFloat(head.Get("Cf-Iplongitude"), 64); errLon == nil && validLatLong(lat, lon) {
			return lat, lon
		}
	}

	if s.locations == nil {
		return 0, 0
	}

	clientIP, ok := httphelper.HeaderClientIP(head)
	if !ok {
		return 0, 0
	}

	location, errLocation := s.locations.LocationByIP(ctx, clientIP)
	if errLocation != nil {
		slog.Debug("Failed to geolocate client ip", slog.String("error", errLocation.Error()))

		return 0, 0
	}

	if !validLatLong(float64(location.Latitude), float64(location.Longitude)) {
		return 0, 0
	}

	return float64(location.Latitude), float64(location.Longitude)
}

func (s Service) State(ctx context.Context, _ *emptypb.Empty) (*v1.StateResponse, error) {
	lat, lon := s.clientLatLong(ctx)
	servers := s.servers.Current()

	for index, srv := range servers {
		servers[index].Distance = float32(distance(float64(srv.Latitude), float64(srv.Longitude), lat, lon))
	}
	sort.Slice(servers, func(i, j int) bool {
		return natural.Less(servers[i].Name, servers[j].Name)
	})

	resp := v1.StateResponse{
		LatLong: &networkv1.LatLong{
			Latitude:  new(float32(lat)),
			Longitude: new(float32(lon)),
		},
	}
	for _, current := range servers {
		resp.Servers = append(resp.Servers, &v1.SafeServer{
			ServerId:      &current.ServerID,
			Host:          &current.Host,
			Port:          new(uint32(current.Port)),
			Ip:            &current.IP,
			Name:          &current.Name,
			NameShort:     &current.NameShort,
			Region:        &current.Region,
			Cc:            &current.CC,
			Players:       &current.Players,
			MaxPlayers:    new(current.MaxPlayerDisplay()),
			Bot:           &current.Bots,
			Map:           &current.Map,
			GameTypes:     current.GameTypes,
			StatsBucketId: &current.StatsBucketID,
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

// Servers returns a list of all servers.
// Public endpoint, no authentication required.
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
		ServerID:             server.GetServerId(),
		ShortName:            server.GetShortName(),
		Name:                 server.GetName(),
		Address:              server.GetAddress(),
		AddressInternal:      server.GetAddressInternal(),
		SDREnabled:           server.GetSdrEnabled(),
		Port:                 uint16(server.GetPort()), //nolint:gosec
		RCON:                 server.GetRcon(),
		Password:             server.GetPassword(),
		IsEnabled:            server.GetIsEnabled(),
		Deleted:              server.GetDeleted(),
		Region:               server.GetRegion(),
		CC:                   server.GetCc(),
		Latitude:             server.GetLatLong().GetLatitude(),
		Longitude:            server.GetLatLong().GetLongitude(),
		LogSecret:            server.GetLogSecret(),
		EnableStats:          server.GetEnableStats(),
		TokenCreatedOn:       server.GetTokenCreatedOn().AsTime(),
		CreatedOn:            server.GetCreatedOn().AsTime(),
		UpdatedOn:            server.GetUpdatedOn().AsTime(),
		DiscordSeedRoleIDs:   server.GetDiscordSeedRoleIds(),
		DiscordSeedChannelID: server.GetDiscordSeedChannelId(),
		IP:                   net.ParseIP(server.GetIp()),
		StatsBucketID:        server.StatsBucketId,
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
		LogSecret:            &server.LogSecret,
		EnableStats:          &server.EnableStats,
		TokenCreatedOn:       timestamppb.New(server.TokenCreatedOn),
		CreatedOn:            timestamppb.New(server.CreatedOn),
		UpdatedOn:            timestamppb.New(server.UpdatedOn),
		DiscordSeedRoleIds:   server.DiscordSeedRoleIDs,
		DiscordSeedChannelId: new(server.DiscordSeedChannelID),
		Ip:                   new(server.IP.String()),
		StatsBucketId:        server.StatsBucketID,
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

// ExecCommand executes an RCON command against the server and returns the
// raw console output.
func (s Service) ExecCommand(ctx context.Context, req *v1.ExecCommandRequest) (*v1.ExecCommandResponse, error) {
	server, errServer := s.servers.Server(ctx, req.GetServerId())
	if errServer != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	command := strings.TrimSpace(req.GetCommand())
	resp, errExec := server.Exec(ctx, command)
	if errExec != nil {
		if errors.Is(errExec, ErrServerDisabled) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, ErrServerDisabled)
		}

		slog.Error("Failed to execute rcon command",
			slog.Int("server_id", int(server.ServerID)),
			slog.String("server", server.ShortName),
			slog.String("command", command),
			slog.String("error", errExec.Error()))

		return nil, connect.NewError(connect.CodeUnavailable, errExec)
	}

	slog.Info("Executed rcon command",
		slog.Int("server_id", int(server.ServerID)),
		slog.String("server", server.ShortName),
		slog.String("command", command),
		slog.String("user", rpc.UserInfoFromCtx(ctx).SteamID.String()))

	return &v1.ExecCommandResponse{Response: &resp}, nil
}

// ClearLogs empties the server's in-memory log buffer. Connected stream
// subscribers keep receiving new live lines afterwards.
func (s Service) ClearLogs(ctx context.Context, req *v1.ClearLogsRequest) (*emptypb.Empty, error) {
	server, errServer := s.servers.Server(ctx, req.GetServerId())
	if errServer != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	s.servers.LogBuffer().Clear(req.GetServerId())

	slog.Info("Cleared server log buffer",
		slog.Int("server_id", int(req.GetServerId())),
		slog.String("server", server.ShortName),
		slog.String("user", rpc.UserInfoFromCtx(ctx).SteamID.String()))

	return &emptypb.Empty{}, nil
}

// StreamLogs sends the server's buffered log history and then streams live log
// lines until the client disconnects.
func (s Service) StreamLogs(ctx context.Context, req *v1.StreamLogsRequest, stream *connect.ServerStream[v1.StreamLogsResponse]) error {
	serverID := req.GetServerId()
	server, errServer := s.servers.Server(ctx, serverID)
	if errServer != nil {
		return connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	buffer := s.servers.LogBuffer()
	sub := make(chan BufferedLog, 512)
	buffer.Consume(serverID, sub)
	defer buffer.Unregister(sub)

	// The snapshot is taken after subscribing, so lines pushed in between are
	// present in both. The seq watermark dedupes them and keeps the stream in
	// order across the history/live boundary.
	snap := buffer.Snapshot(serverID)

	var watermark uint64
	if len(snap) > 0 {
		watermark = snap[len(snap)-1].Seq
	}
	if limit := int(req.GetHistoryLimit()); limit > 0 && len(snap) > limit {
		snap = snap[len(snap)-limit:]
	}

	send := func(line BufferedLog) error {
		return stream.Send(line.toStreamLogsResponse(serverID, server.ShortName))
	}

	for _, line := range snap {
		if errSend := send(line); errSend != nil {
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-sub:
			if !ok {
				return nil
			}
			if line.Seq <= watermark {
				continue
			}
			watermark = line.Seq
			if errSend := send(line); errSend != nil {
				return nil
			}
		}
	}
}

func (l BufferedLog) toStreamLogsResponse(serverID int32, serverName string) *v1.StreamLogsResponse {
	//nolint:gosec // logparse.EventType values are bounded well below int32 range
	eventType := v1.LogEventType(int32(l.EventType))

	return &v1.StreamLogsResponse{
		ServerId:   &serverID,
		ServerName: &serverName,
		EventType:  &eventType,
		Body:       &l.Raw,
		CreatedOn:  timestamppb.New(l.CreatedOn),
	}
}
