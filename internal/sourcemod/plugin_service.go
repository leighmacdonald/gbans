package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	banv1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	v1 "github.com/leighmacdonald/gbans/internal/sourcemod/v1"
	"github.com/leighmacdonald/gbans/internal/sourcemod/v1/sourcemodv1connect"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
)

var errTooOften = errors.New("you must wait before trying to mod ping again")

type TokenGeneratorFn func(serverID int32, serverName string) (string, error)

// PluginService is responsible for all communication with the game servers using the sourcemod plugin.
type PluginService struct {
	sourcemod               Sourcemod
	notifier                notification.Notifier
	persons                 *person.Persons
	serverAuth              rpc.ServerAuthenticator
	tokenGenerator          TokenGeneratorFn
	evades                  EvadeChecker
	logChannelID            string
	pingHistory             map[steamid.SteamID]time.Time
	pingHistoryMu           *sync.Mutex
	minPingModRetryInterval time.Duration
}

func NewPluginService(sourcemod Sourcemod, persons *person.Persons, serverAuthenticator rpc.ServerAuthenticator, evades EvadeChecker, tokenGenerator TokenGeneratorFn, notifier notification.Notifier, logChannelID string, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := sourcemodv1connect.NewPluginServiceHandler(PluginService{
		sourcemod:               sourcemod,
		persons:                 persons,
		tokenGenerator:          tokenGenerator,
		notifier:                notifier,
		evades:                  evades,
		logChannelID:            logChannelID,
		serverAuth:              serverAuthenticator,
		pingHistory:             map[steamid.SteamID]time.Time{},
		pingHistoryMu:           &sync.Mutex{},
		minPingModRetryInterval: time.Minute * 5,
	}, option...)

	serverAuth := rpc.NewServerAuthenticator()

	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMCheckProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMOverridesProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMUsersProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMGroupsProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMSeedProcedure, serverAuth)

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s PluginService) SMPingMod(ctx context.Context, req *v1.SMPingModRequest) (*emptypb.Empty, error) {
	serverInfo := rpc.ServerInfoFromCtx(ctx)
	steamID := steamid.New(req.GetSteamId())

	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	s.pingHistoryMu.Lock()
	defer s.pingHistoryMu.Unlock()
	lastTry, ok := s.pingHistory[steamID]
	if ok && time.Since(lastTry) < s.minPingModRetryInterval {
		return nil, connect.NewError(connect.CodeResourceExhausted, errTooOften)
	}

	if err := s.sourcemod.PingMod(ctx, steamID, req.GetName(), req.GetReason(), req.GetClientId(), serverInfo.ServerName); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	s.pingHistory[steamID] = time.Now()

	return &emptypb.Empty{}, nil
}

func (s PluginService) SMAuthenticate(ctx context.Context, req *v1.SMAuthenticateRequest) (*v1.SMAuthenticateResponse, error) {
	password := req.GetPassword()
	if password == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	serverID, name, err := s.serverAuth.GetByPassword(ctx, password)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	token, errToken := s.tokenGenerator(serverID, name)
	if errToken != nil || token == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	slog.Debug("Server authentication", slog.Int("serverId", int(serverID)), slog.String("name", name))

	return &v1.SMAuthenticateResponse{Token: &token}, nil
}

// SMCheck verifies whether a connecting player is banned by their SteamID or IP address.
// It fails open on errors, allowing the player to connect rather than causing a denial-of-service.
// If a ban is found that doesn't match the provided SteamID directly, it checks for evasion
// bans before returning the result. Discord notifications are sent asynchronously on denials.
func (s PluginService) SMCheck(ctx context.Context, req *v1.SMCheckRequest) (*v1.SMCheckResponse, error) {
	defaultResponse := &v1.SMCheckResponse{
		ClientId: new(req.GetClientId()),
		BanType:  new(banv1.BanType_BAN_TYPE_OK_UNSPECIFIED),
		Msg:      new(""),
	}
	steamID := steamid.New(req.GetSteamId())
	ipAddr, errIP := netip.ParseAddr(req.GetIp())
	if errIP != nil {
		slog.Error("Failed to parse IP", slog.String("error", errIP.Error()))

		return defaultResponse, nil //nolint:nilerr
	}

	banState, msg, errBS := s.sourcemod.GetBanState(ctx, steamID, ipAddr)
	if errBS != nil {
		slog.Error("failed to get ban state", slog.String("error", errBS.Error()))

		// Fail Open
		return defaultResponse, nil //nolint:nilerr
	}

	if banState.BanID == 0 {
		slog.Debug("Player connect check", slog.String("steam_id", steamID.String()), slog.Bool("success", true))

		return defaultResponse, nil
	}

	if errPlayer := s.persons.EnsurePerson(ctx, steamID); errPlayer != nil {
		slog.Error("Failed to load or create player on connect")

		return defaultResponse, nil //nolint:nilerr
	}

	if banState.SteamID != steamID && !banState.EvadeOK {
		evadeBanned, err := s.evades.CheckEvadeStatus(ctx, steamID, ipAddr)
		if err != nil {
			slog.Error("Failed to check evade status", slog.String("error", err.Error()))

			return defaultResponse, nil //nolint:nilerr
		}

		if evadeBanned {
			go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

			return &v1.SMCheckResponse{ClientId: defaultResponse.ClientId, BanType: toBanType(bantype.Banned), Msg: new("Evasion ban")}, nil
		}
	}

	if banState.SteamID != steamID && banState.EvadeOK {
		return defaultResponse, nil
	}

	go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

	return &v1.SMCheckResponse{
		ClientId: defaultResponse.ClientId,
		BanType:  toBanType(banState.BanType),
		Msg:      &msg,
	}, nil
}

func (s PluginService) SMOverrides(ctx context.Context, _ *emptypb.Empty) (*v1.SMOverridesResponse, error) {
	overrides, errOverrides := s.sourcemod.Overrides(ctx)
	if errOverrides != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMOverridesResponse{Overrides: make([]*v1.SMOverride, len(overrides))}
	for idx, override := range overrides {
		resp.Overrides[idx] = &v1.SMOverride{
			OverrideType: toOverrideType(override.Type),
			Name:         &override.Name,
			Flags:        &override.Flags,
		}
	}

	return &resp, nil
}

func (s PluginService) SMUsers(ctx context.Context, _ *emptypb.Empty) (*v1.SMUsersResponse, error) {
	users, errUsers := s.sourcemod.Admins(ctx)
	if errUsers != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMUsersResponse{
		Users:      make([]*v1.SMUser, len(users)),
		UserGroups: nil,
	}

	for idx, user := range users {
		resp.Users[idx] = &v1.SMUser{
			Id:       &user.AdminID,
			AuthType: toAuthType(user.AuthType),
			Identity: &user.Identity,
			Password: &user.Password,
			Flags:    &user.Flags,
			Name:     &user.Name,
			Immunity: &user.Immunity,
		}

		for _, ug := range user.Groups {
			resp.UserGroups = append(resp.UserGroups, &v1.SMUserGroup{
				AdminId:   &user.AdminID,
				GroupName: &ug.Name,
			})
		}
	}

	return &resp, nil
}

func (s PluginService) SMGroups(ctx context.Context, _ *emptypb.Empty) (*v1.SMGroupsResponse, error) {
	groups, errGroups := s.sourcemod.Groups(ctx)
	if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
	if errImmunities != nil && !errors.Is(errImmunities, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMGroupsResponse{
		Groups:     make([]*v1.Group, len(groups)),
		Immunities: make([]*v1.SMGroupImmunity, len(immunities)),
	}

	//goland:noinspection ALL
	for idx, group := range groups {
		resp.Groups[idx] = &v1.Group{
			Flags:         &group.Flags,
			Name:          &group.Name,
			ImmunityLevel: &group.ImmunityLevel,
		}
	}

	for idx, immunity := range immunities {
		resp.Immunities[idx] = &v1.SMGroupImmunity{
			GroupName: &immunity.Group.Name,
			OtherName: &immunity.Other.Name,
		}
	}

	if resp.Groups == nil {
		resp.Groups = []*v1.Group{}
	}

	if resp.Immunities == nil {
		resp.Immunities = []*v1.SMGroupImmunity{}
	}

	return &resp, nil
}

func (s PluginService) SMSeed(ctx context.Context, req *v1.SMSeedRequest) (*v1.SMSeedResponse, error) {
	serverInfo := rpc.ServerInfoFromCtx(ctx)
	steamID := steamid.New(req.GetSteamId())
	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	server, errServer := s.sourcemod.servers.Server(ctx, serverInfo.ServerID)
	if errServer != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	var serverState servers.SafeServer
	for _, srv := range s.sourcemod.servers.Current() {
		if serverInfo.ServerID == server.ServerID {
			serverState = srv

			break
		}
	}

	if serverState.ServerID == 0 {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if !s.sourcemod.seedRequest(server.DiscordSeedRoleIDs, serverState, steamID.String()) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.SMSeedResponse{Message: new("Successfully sent request")}, nil
}
