package roles

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/roles/v1/rolesv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	roles Roles
}

func NewService(roles Roles, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := rolesv1connect.NewRolesServiceHandler(Service{roles: roles}, option...)

	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleListProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleCreateProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleEditProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleDeleteProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleAssignProcedure, rpc.WithMinPermissions(permission.Admin))
	authMiddleware.UserRoute(rolesv1connect.RolesServiceRoleBySteamIDProcedure, rpc.WithMinPermissions(permission.Admin))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) RoleList(ctx context.Context, _ *emptypb.Empty) (*rolesv1.RoleListResponse, error) {
	allRoles, err := s.roles.GetAll(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &rolesv1.RoleListResponse{Roles: make([]*rolesv1.Role, len(allRoles))}
	for idx, role := range allRoles {
		resp.Roles[idx] = toProtoRole(role)
	}

	return resp, nil
}

func (s Service) RoleCreate(ctx context.Context, req *rolesv1.RoleCreateRequest) (*rolesv1.RoleCreateResponse, error) {
	role, errSave := s.roles.Create(ctx, req.GetRoleName(), permissionsToStrings(req.GetPermissions()))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("Role created", slog.String("role_name", role.RoleName), slog.Int("role_id", int(role.RoleID)))

	return &rolesv1.RoleCreateResponse{Role: toProtoRole(role)}, nil
}

func (s Service) RoleEdit(ctx context.Context, req *rolesv1.RoleEditRequest) (*rolesv1.RoleEditResponse, error) {
	role, errUpdate := s.roles.Edit(ctx, req.GetRoleId(), req.GetRoleName(), permissionsToStrings(req.GetPermissions()))
	if errUpdate != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("Role updated", slog.String("role_name", role.RoleName), slog.Int("role_id", int(role.RoleID)))

	return &rolesv1.RoleEditResponse{}, nil
}

func (s Service) RoleDelete(ctx context.Context, req *rolesv1.RoleDeleteRequest) (*emptypb.Empty, error) {
	if err := s.roles.Delete(ctx, int32(req.GetRoleId())); err != nil { //nolint:gosec
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("Role deleted", slog.Int("role_id", int(req.GetRoleId())))

	return &emptypb.Empty{}, nil
}

func (s Service) RoleAssign(ctx context.Context, req *rolesv1.RoleAssignRequest) (*emptypb.Empty, error) {
	if err := s.roles.Assign(ctx, steamid.New(req.GetSteamId()), req.GetRoleId()); err != nil { //nolint:gosec
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("Role assigned", slog.Int64("steam_id", int64(req.GetSteamId())), slog.Int("role_id", int(req.GetRoleId()))) //nolint:gosec

	return &emptypb.Empty{}, nil
}

func (s Service) RoleBySteamID(ctx context.Context, req *rolesv1.RoleBySteamIDRequest) (*rolesv1.RoleBySteamIDResponse, error) {
	assignedRoles, err := s.roles.GetRolesBySteamID(ctx, steamid.New(req.GetSteamId())) //nolint:gosec
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := &rolesv1.RoleBySteamIDResponse{Roles: make([]*rolesv1.Role, len(assignedRoles))}
	for idx, role := range assignedRoles {
		resp.Roles[idx] = toProtoRole(role)
	}

	return resp, nil
}

func toProtoRole(role Role) *rolesv1.Role {
	return &rolesv1.Role{
		RoleId:      &role.RoleID,
		RoleName:    &role.RoleName,
		Permissions: stringsToPermissions(role.Permissions),
		CreatedOn:   timestamppb.New(role.CreatedOn),
		UpdatedOn:   timestamppb.New(role.UpdatedOn),
		UserCount:   &role.UserCount,
	}
}

func stringsToPermissions(perms []string) []rolesv1.Permission {
	out := make([]rolesv1.Permission, 0, len(perms))

	for _, s := range perms {
		p, ok := rolesv1.Permission_value[s]
		if !ok || p == 0 {
			continue
		}

		out = append(out, rolesv1.Permission(p))
	}

	if out == nil {
		return []rolesv1.Permission{}
	}

	return out
}

func permissionsToStrings(perms []rolesv1.Permission) []string {
	out := make([]string, 0, len(perms))

	for _, p := range perms {
		if p == rolesv1.Permission_PERMISSION_UNSPECIFIED {
			continue
		}

		out = append(out, p.String())
	}

	return out
}
