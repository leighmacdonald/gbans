package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/sourcemod/v1"
	"github.com/leighmacdonald/gbans/internal/sourcemod/v1/sourcemodv1connect"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EvadeChecker interface {
	CheckEvadeStatus(ctx context.Context, steamID steamid.SteamID, address netip.Addr) (bool, error)
}

type Service struct {
	sourcemodv1connect.UnimplementedSourcemodServiceHandler

	sourcemod    Sourcemod
	notifier     notification.Notifier
	persons      *person.Persons
	evades       EvadeChecker
	logChannelID string
}

func NewService(sourcemod Sourcemod, persons *person.Persons, notifier notification.Notifier, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := sourcemodv1connect.NewSourcemodServiceHandler(Service{sourcemod: sourcemod, persons: persons, notifier: notifier}, option...)

	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceGroupsProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCreateGroupProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceEditGroupsProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceDeleteGroupProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceGroupOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCreateGroupOverrideProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceEditGroupOverrideProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceDeleteGroupOverrideProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceAdminsProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCreateAdminProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceEditAdminProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceDeleteAdminProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceAddAdminGroupProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceDeleteAdminGroupProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCreateOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceEditOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceDeleteOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceGroupImmunitiesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCreateImmunityProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceCheckProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceSMOverridesProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceSMUsersProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceSMGroupsProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.AuthedRoute(sourcemodv1connect.SourcemodServiceSMSeedProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Groups(ctx context.Context, _ *emptypb.Empty) (*v1.GroupsResponse, error) {
	groups, errGroups := s.sourcemod.Groups(ctx)
	if errGroups != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GroupsResponse{Groups: make([]*v1.Group, len(groups))}
	for idx, grp := range groups {
		resp.Groups[idx] = toGroup(grp)
	}

	return &resp, nil
}

func toGroup(group Groups) *v1.Group {
	return &v1.Group{
		GroupId:       &group.GroupID,
		Flags:         &group.Flags,
		Name:          &group.Name,
		ImmunityLevel: &group.ImmunityLevel,
		CreatedOn:     timestamppb.New(group.CreatedOn),
		UpdatedOn:     timestamppb.New(group.UpdatedOn),
	}
}

func (s Service) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.CreateGroupResponse, error) {
	group, errGroup := s.sourcemod.AddGroup(ctx, req.GetName(), req.GetFlags(), req.GetImmunity())
	if errGroup != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateGroupResponse{Group: toGroup(group)}, nil
}

func (s Service) EditGroups(ctx context.Context, req *v1.EditGroupsRequest) (*v1.EditGroupsResponse, error) {
	group, errGroup := s.sourcemod.GetGroupByID(ctx, req.GetGroupId())
	if errGroup != nil {
		if errors.Is(errGroup, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	group.Name = req.GetName()
	group.Flags = req.GetFlags()
	group.ImmunityLevel = req.GetImmunity()

	editedGroup, errSave := s.sourcemod.SaveGroup(ctx, group)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditGroupsResponse{Group: toGroup(editedGroup)}, nil
}

func (s Service) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest) (*emptypb.Empty, error) {
	if err := s.sourcemod.DelGroup(ctx, req.GetGroupId()); err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) GroupOverrides(ctx context.Context, req *v1.GroupOverridesRequest) (*v1.GroupOverridesResponse, error) {
	overrides, errOverrides := s.sourcemod.GroupOverrides(ctx, req.GetGroupId())
	if errOverrides != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GroupOverridesResponse{Overrides: make([]*v1.GroupOverrides, len(overrides))}
	for idx, override := range overrides {
		resp.Overrides[idx] = &v1.GroupOverrides{
			GroupOverrideId: &override.GroupOverrideID,
			GroupId:         &override.GroupID,
			OverrideType:    toOverrideType(override.Type),
			Name:            &override.Name,
			OverrideAccess:  toOverrideAccess(override.Access),
			CreatedOn:       timestamppb.New(override.CreatedOn),
			UpdatedOn:       timestamppb.New(override.UpdatedOn),
		}
	}

	return &resp, nil
}

func (s Service) CreateGroupOverride(ctx context.Context, req *v1.CreateGroupOverrideRequest) (*v1.CreateGroupOverrideResponse, error) {
	override, errOverride := s.sourcemod.AddGroupOverride(ctx, req.GetGroupId(), req.GetName(),
		fromOverrideType(req.GetType()),
		fromOverrideAccess(req.GetAccess()))
	if errOverride != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateGroupOverrideResponse{GroupOverride: &v1.GroupOverrides{
		GroupOverrideId: &override.GroupOverrideID,
		GroupId:         &override.GroupID,
		OverrideType:    toOverrideType(override.Type),
		Name:            &override.Name,
		OverrideAccess:  toOverrideAccess(override.Access),
		CreatedOn:       timestamppb.New(override.CreatedOn),
		UpdatedOn:       timestamppb.New(override.UpdatedOn),
	}}, nil
}

func (s Service) EditGroupOverride(ctx context.Context, req *v1.EditGroupOverrideRequest) (*v1.EditGroupOverrideResponse, error) {
	override, errOverride := s.sourcemod.GroupOverride(ctx, req.GetGroupOverrideId())
	if errOverride != nil {
		if errors.Is(errOverride, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	override.Type = fromOverrideType(req.GetOverrideType())
	override.Name = req.GetName()
	override.Access = fromOverrideAccess(req.GetOverrideAccess())

	edited, errSave := s.sourcemod.SaveGroupOverride(ctx, override)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditGroupOverrideResponse{GroupOverride: &v1.GroupOverrides{
		GroupOverrideId: &edited.GroupOverrideID,
		GroupId:         &edited.GroupID,
		OverrideType:    toOverrideType(edited.Type),
		Name:            &edited.Name,
		OverrideAccess:  toOverrideAccess(edited.Access),
		CreatedOn:       timestamppb.New(edited.CreatedOn),
		UpdatedOn:       timestamppb.New(edited.UpdatedOn),
	}}, nil
}

func (s Service) DeleteGroupOverride(ctx context.Context, req *v1.DeleteGroupOverrideRequest) (*emptypb.Empty, error) {
	if err := s.sourcemod.DelGroupOverride(ctx, req.GetGroupOverrideId()); err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Admins(ctx context.Context, _ *emptypb.Empty) (*v1.AdminsResponse, error) {
	admins, errAdmins := s.sourcemod.Admins(ctx)
	if errAdmins != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.AdminsResponse{Admins: make([]*v1.Admin, len(admins))}
	for idx, admin := range admins {
		resp.Admins[idx] = toAdmin(admin)
	}

	return &resp, nil
}

func (s Service) CreateAdmin(ctx context.Context, req *v1.CreateAdminRequest) (*v1.CreateAdminResponse, error) {
	admin, errAdmin := s.sourcemod.AddAdmin(ctx, req.GetName(), fromAuthType(req.GetAuthType()), req.GetIdentity(), req.GetFlags(), req.GetImmunity(), req.GetPassword())
	if errAdmin != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateAdminResponse{Admin: toAdmin(admin)}, nil
}

func (s Service) EditAdmin(ctx context.Context, req *v1.EditAdminRequest) (*v1.EditAdminResponse, error) {
	admin, errAdmin := s.sourcemod.AdminByID(ctx, req.GetAdminId())
	if errAdmin != nil {
		if errors.Is(errAdmin, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	admin.Name = req.GetName()
	admin.Flags = req.GetFlags()
	admin.Immunity = req.GetImmunity()
	admin.AuthType = fromAuthType(req.GetAuthType())
	admin.Identity = req.GetIdentity()
	admin.Password = req.GetPassword()

	edited, errSave := s.sourcemod.SaveAdmin(ctx, admin)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditAdminResponse{Admin: toAdmin(edited)}, nil
}

func (s Service) DeleteAdmin(ctx context.Context, req *v1.DeleteAdminRequest) (*emptypb.Empty, error) {
	if err := s.sourcemod.DelAdmin(ctx, req.GetAdminId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) AddAdminGroup(ctx context.Context, req *v1.AddAdminGroupRequest) (*v1.AddAdminGroupResponse, error) {
	adminGroup, err := s.sourcemod.AddAdminGroup(ctx, req.GetAdminId(), req.GetGroupId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.AddAdminGroupResponse{Admin: toAdmin(adminGroup)}, nil
}

func (s Service) DeleteAdminGroup(ctx context.Context, req *v1.DeleteAdminGroupRequest) (*emptypb.Empty, error) {
	if _, errDel := s.sourcemod.DelAdminGroup(ctx, req.GetAdminId(), req.GetGroupId()); errDel != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Overrides(ctx context.Context, _ *emptypb.Empty) (*v1.OverridesResponse, error) {
	overrides, errOverrides := s.sourcemod.Overrides(ctx)
	if errOverrides != nil && !errors.Is(errOverrides, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.OverridesResponse{Overrides: make([]*v1.Override, len(overrides))}

	for idx, override := range overrides {
		resp.Overrides[idx] = toOverride(override)
	}

	return &resp, nil
}

func (s Service) CreateOverrides(ctx context.Context, req *v1.CreateOverridesRequest) (*v1.CreateOverridesResponse, error) {
	override, errCreate := s.sourcemod.AddOverride(ctx, req.GetName(), fromOverrideType(req.GetOverrideType()), req.GetFlags())
	if errCreate != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateOverridesResponse{Override: toOverride(override)}, nil
}

func (s Service) EditOverrides(ctx context.Context, req *v1.EditOverridesRequest) (*v1.EditOverridesResponse, error) {
	override, errOverride := s.sourcemod.Override(ctx, req.GetOverrideId())
	if errOverride != nil {
		if errors.Is(errOverride, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	override.Type = fromOverrideType(req.GetOverrideType())
	override.Name = req.GetName()
	override.Flags = req.GetFlags()

	edited, errSave := s.sourcemod.SaveOverride(ctx, override)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditOverridesResponse{Override: toOverride(edited)}, nil
}

func (s Service) DeleteOverrides(ctx context.Context, req *v1.DeleteOverridesRequest) (*emptypb.Empty, error) {
	if errCreate := s.sourcemod.DelOverride(ctx, req.GetOverrideId()); errCreate != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) GroupImmunities(ctx context.Context, _ *emptypb.Empty) (*v1.GroupImmunitiesResponse, error) {
	immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
	if errImmunities != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GroupImmunitiesResponse{GroupImmunities: make([]*v1.GroupImmunity, len(immunities))}
	for idx, immunity := range immunities {
		resp.GroupImmunities[idx] = toGroupImmunity(immunity)
	}

	return &resp, nil
}

func (s Service) CreateImmunity(ctx context.Context, req *v1.CreateImmunityRequest) (*v1.CreateImmunityResponse, error) {
	immunity, errImmunity := s.sourcemod.AddGroupImmunity(ctx, req.GetGroupId(), req.GetOtherId())
	if errImmunity != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateImmunityResponse{GroupImmunity: toGroupImmunity(immunity)}, nil
}

func (s Service) DeleteImmunity(ctx context.Context, req *v1.DeleteImmunityRequest) (*emptypb.Empty, error) {
	if err := s.sourcemod.DelGroupImmunity(ctx, req.GetImmunityId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Check(ctx context.Context, req *v1.CheckRequest) (*v1.CheckResponse, error) {
	defaultResponse := &v1.CheckResponse{
		ClientId: ptr.To(req.GetClientId()),
		BanType:  ptr.To(v1.BanType_BAN_TYPE_OK_UNSPECIFIED),
		Msg:      ptr.To(""),
	}
	steamID := steamid.New(req.GetSteamId())
	// steamID, valid := req.SteamID
	// if !valid {
	// 	ctx.JSON(http.StatusOK, defaultValue)
	// 	slog.Error("Did not receive valid steamid for check response", log.ErrAttr(steamid.ErrDecodeSID))

	// 	return
	// }

	ipAddr, errIP := netip.ParseAddr(req.GetIp())
	if errIP != nil {
		slog.Error("Failed to parse IP", slog.String("error", errIP.Error()))

		return defaultResponse, nil
	}

	banState, msg, errBS := s.sourcemod.GetBanState(ctx, steamID, ipAddr)
	if errBS != nil {
		slog.Error("failed to get ban state", slog.String("error", errBS.Error()))

		// Fail Open
		return defaultResponse, nil
	}

	if banState.BanID == 0 {
		return defaultResponse, nil
	}

	if errPlayer := s.persons.EnsurePerson(ctx, steamID); errPlayer != nil {
		slog.Error("Failed to load or create player on connect")

		return defaultResponse, nil
	}

	if banState.SteamID != steamID && !banState.EvadeOK {
		evadeBanned, err := s.evades.CheckEvadeStatus(ctx, steamID, ipAddr)
		if err != nil {
			return defaultResponse, nil
		}

		if evadeBanned {
			go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

			return &v1.CheckResponse{ClientId: defaultResponse.ClientId, BanType: toBanType(bantype.Banned), Msg: ptr.To("Evasion ban")}, nil
		}
	}

	if banState.SteamID != steamID && banState.EvadeOK {
		return defaultResponse, nil
	}

	go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

	return &v1.CheckResponse{
		ClientId: defaultResponse.ClientId,
		BanType:  toBanType(banState.BanType),
		Msg:      &msg,
	}, nil
}

func (s Service) SMOverrides(ctx context.Context, _ *emptypb.Empty) (*v1.SMOverridesResponse, error) {
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

func (s Service) SMUsers(ctx context.Context, _ *emptypb.Empty) (*v1.SMUsersResponse, error) {
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
			Id:       nil,
			AuthType: nil,
			Identity: nil,
			Password: nil,
			Flags:    nil,
			Name:     nil,
			Immunity: nil,
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

func (s Service) SMGroups(ctx context.Context, _ *emptypb.Empty) (*v1.SMGroupsResponse, error) {
	groups, errGroups := s.sourcemod.Groups(ctx)
	if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
	if errImmunities != nil && !errors.Is(errImmunities, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMGroupsResponse{
		Groups:     make([]*v1.SMGroup, len(groups)),
		Immunities: make([]*v1.SMGroupImmunity, len(immunities)),
	}

	//goland:noinspection ALL
	for idx, group := range groups {
		resp.Groups[idx] = &v1.SMGroup{
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

	return &resp, nil
}

func (s Service) SMSeed(ctx context.Context, req *v1.SMSeedRequest) (*v1.SMSeedResponse, error) {
	serverInfo, _ := rpc.ServerInfoFromCtx(ctx)
	// FIXME
	steamID := steamid.New(req.GetSteamId())
	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	server, errServer := s.sourcemod.servers.Server(ctx, serverInfo.ServerID)
	if errServer != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	if !s.sourcemod.seedRequest(ctx, server, steamID.String()) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.SMSeedResponse{Message: ptr.To("Successfully sent request")}, nil
}

func toOverrideType(override OverrideType) *v1.OverrideType {
	overrideType := v1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED
	if override == OverrideTypeGroup {
		overrideType = v1.OverrideType_OVERRIDE_TYPE_GROUP
	}

	return &overrideType
}

func toOverrideAccess(override OverrideAccess) *v1.OverrideAccess {
	access := v1.OverrideAccess_OVERRIDE_ACCESS_ALLOW_UNSPECIFIED
	if override == OverrideAccessDeny {
		access = v1.OverrideAccess_OVERRIDE_ACCESS_DENY
	}

	return &access
}

func fromOverrideType(overrideType v1.OverrideType) OverrideType {
	switch overrideType {
	case v1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED:
		return OverrideTypeCommand
	default:
		return OverrideTypeGroup
	}
}

func fromOverrideAccess(access v1.OverrideAccess) OverrideAccess {
	switch access {
	case v1.OverrideAccess_OVERRIDE_ACCESS_ALLOW_UNSPECIFIED:
		return OverrideAccessAllow
	case v1.OverrideAccess_OVERRIDE_ACCESS_DENY:
		return OverrideAccessDeny
	default:
		return OverrideAccessAllow
	}
}

func toAuthType(authType AuthType) *v1.AuthType {
	switch authType {
	case AuthTypeName:
		return ptr.To(v1.AuthType_AUTH_TYPE_NAME)
	case AuthTypeIP:
		return ptr.To(v1.AuthType_AUTH_TYPE_IP)
	case AuthTypeSteam:
		fallthrough
	default:
		return ptr.To(v1.AuthType_AUTH_TYPE_STEAM_UNSPECIFIED)
	}
}

func fromAuthType(authType v1.AuthType) AuthType {
	switch authType {
	case v1.AuthType_AUTH_TYPE_NAME:
		return AuthTypeName
	case v1.AuthType_AUTH_TYPE_IP:
		return AuthTypeIP
	case v1.AuthType_AUTH_TYPE_STEAM_UNSPECIFIED:
		fallthrough
	default:
		return AuthTypeSteam
	}
}

func toOverride(override Overrides) *v1.Override {
	return &v1.Override{
		OverrideId:   &override.OverrideID,
		OverrideType: toOverrideType(override.Type),
		Name:         &override.Name,
		Flags:        &override.Flags,
		CreatedOn:    timestamppb.New(override.CreatedOn),
		UpdatedOn:    timestamppb.New(override.UpdatedOn),
	}
}

func toAdmin(admin Admin) *v1.Admin {
	resp := v1.Admin{
		AdminId:   &admin.AdminID,
		SteamId:   ptr.To(admin.SteamID.Int64()),
		AuthType:  toAuthType(admin.AuthType),
		Identity:  &admin.Identity,
		Password:  &admin.Password,
		Flags:     &admin.Flags,
		Name:      &admin.Name,
		Immunity:  &admin.Immunity,
		Groups:    make([]*v1.Group, len(admin.Groups)),
		CreatedOn: timestamppb.New(admin.CreatedOn),
		UpdatedOn: timestamppb.New(admin.UpdatedOn),
	}

	for idx, group := range admin.Groups {
		resp.Groups[idx] = toGroup(group)
	}

	return &resp
}

func toGroupImmunity(immunity GroupImmunity) *v1.GroupImmunity {
	return &v1.GroupImmunity{
		GroupImmunityId: &immunity.GroupImmunityID,
		Group:           toGroup(immunity.Group),
		Other:           toGroup(immunity.Other),
		CreatedOn:       timestamppb.New(immunity.CreatedOn),
	}
}

func toBanType(banType bantype.Type) *v1.BanType {
	switch banType {
	case bantype.NoComm:
		return ptr.To(v1.BanType_BAN_TYPE_NO_COMM)
	case bantype.Network:
		return ptr.To(v1.BanType_BAN_TYPE_NETWORK)
	case bantype.Banned:
		return ptr.To(v1.BanType_BAN_TYPE_BANNED)
	case bantype.Unknown:
		return ptr.To(v1.BanType_BAN_TYPE_OK_UNSPECIFIED)
	default:
		return ptr.To(v1.BanType_BAN_TYPE_OK_UNSPECIFIED)
	}
}
