package sourcemod_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	banv1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/roles"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	sourcemodv1 "github.com/leighmacdonald/gbans/internal/sourcemod/v1"
	"github.com/leighmacdonald/gbans/internal/sourcemod/v1/sourcemodv1connect"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

// rpcHarness wires the sourcemod connect services behind the real auth
// middleware and serves them over a local httptest server.
type rpcHarness struct {
	sm      sourcemod.Sourcemod
	servers *servers.Servers
	bans    ban.Bans
	roleSvc *roles.Roles
	mw      *rpc.Middleware
	server  *httptest.Server
}

func newRPCHarness(t *testing.T) *rpcHarness {
	t.Helper()

	roleSvc := roles.NewRoles(roles.NewRepository(fixture.Database), tests.OwnerSID)
	roleAuth := rpc.NewRoleAuth(roleSvc)
	middleware := rpc.NewMiddleware("gbans-test", "test-cookie-secret", roleSvc)

	serversCase, errServers := servers.New(servers.NewRepository(fixture.Database), nil, "")
	require.NoError(t, errServers)

	smSvc := sourcemod.New(
		sourcemod.NewRepository(fixture.Database),
		fixture.Persons,
		notification.NewDiscard(),
		"seed-channel",
		"log-channel",
		"mod-role",
		serversCase,
	)

	// A zero Reports value is fine: none of the sourcemod service tests attach
	// bans to reports, and that is the only path that touches it.
	bansCase := ban.New(
		ban.NewRepository(fixture.Database),
		fixture.Persons,
		"log-channel",
		"",
		tests.OwnerSID,
		ban.Reports{},
		notification.NewDiscard(),
		serversCase,
		tests.EmptyIPProvider{},
	)

	webSvc := sourcemod.NewSourcemodService(smSvc, roleAuth, middleware)
	pluginSvc := sourcemod.NewPluginService(
		smSvc,
		person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi),
		serversCase,
		bansCase,
		rpc.NewServerTokenGenerator("gbans-test", []byte("test-cookie-secret")),
		notification.NewDiscard(),
		"log-channel",
		middleware,
	)

	api := http.NewServeMux()
	for _, svc := range []rpc.Service{webSvc, pluginSvc} {
		api.Handle(svc.Pattern, svc.Handler)
	}

	// Register the discord message templates so denial/seed rendering works
	// outside the discord-gated app wiring.
	sourcemod.RegisterDiscordCommands(discord.Discard{}, smSvc, serversCase)

	handler := authn.NewMiddleware(middleware.Authenticate).Wrap(api)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &rpcHarness{
		sm: smSvc, servers: serversCase, bans: bansCase, roleSvc: roleSvc, mw: middleware, server: server,
	}
}

// bearerOption returns a client option that attaches the bearer token and,
// when non-empty, the fingerprint cookie to every request.
func bearerOption(token, fingerprint string) connect.ClientOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if token != "" {
				req.Header().Set("Authorization", "Bearer "+token)
			}

			if fingerprint != "" {
				req.Header().Set("Cookie", "fingerprint="+fingerprint)
			}

			return next(ctx, req)
		}
	}))
}

func requireCode(t *testing.T, want connect.Code, err error) {
	t.Helper()

	require.Error(t, err)
	require.Equal(t, want, connect.CodeOf(err))
}

func (harness *rpcHarness) webClient(t *testing.T, person personDomain.Core) sourcemodv1connect.SourcemodServiceClient {
	t.Helper()

	token, fingerprint, err := harness.mw.MakeUserToken(person)
	require.NoError(t, err)

	return sourcemodv1connect.NewSourcemodServiceClient(harness.server.Client(), harness.server.URL, bearerOption(token, fingerprint))
}

func (harness *rpcHarness) anonWebClient() sourcemodv1connect.SourcemodServiceClient {
	return sourcemodv1connect.NewSourcemodServiceClient(harness.server.Client(), harness.server.URL)
}

func (harness *rpcHarness) pluginClient(t *testing.T, token string) sourcemodv1connect.PluginServiceClient {
	t.Helper()

	return sourcemodv1connect.NewPluginServiceClient(harness.server.Client(), harness.server.URL, bearerOption(token, ""))
}

func (harness *rpcHarness) anonPluginClient() sourcemodv1connect.PluginServiceClient {
	return sourcemodv1connect.NewPluginServiceClient(harness.server.Client(), harness.server.URL)
}

// grantRole creates a role with the given permission enum names and assigns it
// to the steam id.
func (harness *rpcHarness) grantRole(t *testing.T, sid steamid.SteamID, perms ...string) {
	t.Helper()

	roleName := "svc-" + stringutil.SecureRandomString(8)
	role, err := harness.roleSvc.Create(t.Context(), roleName, perms)
	require.NoError(t, err)
	require.NoError(t, harness.roleSvc.Assign(t.Context(), sid, role.RoleID))
}

// serverToken registers a new test server and authenticates it, returning the
// server JWT.
func (harness *rpcHarness) serverToken(t *testing.T) string {
	t.Helper()

	server := fixture.CreateTestServer(t.Context())

	resp, err := harness.anonPluginClient().SMAuthenticate(t.Context(), &sourcemodv1.SMAuthenticateRequest{
		Password: &server.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetToken())

	return resp.GetToken()
}

// seedServer registers a test server with discord seed roles configured and
// returns its JWT.
func (harness *rpcHarness) seedServerToken(t *testing.T) string {
	t.Helper()

	server, err := harness.servers.Save(t.Context(), servers.Server{
		Name:               stringutil.SecureRandomString(10),
		ShortName:          stringutil.SecureRandomString(3),
		Address:            "192.0.2.10",
		Password:           stringutil.SecureRandomString(10),
		LogSecret:          87654321,
		Port:               27015,
		RCON:               stringutil.SecureRandomString(10),
		IsEnabled:          true,
		Region:             "eu",
		CreatedOn:          time.Now(),
		UpdatedOn:          time.Now(),
		DiscordSeedRoleIDs: []string{"1234567890123456789"},
	})
	require.NoError(t, err)

	resp, err := harness.anonPluginClient().SMAuthenticate(t.Context(), &sourcemodv1.SMAuthenticateRequest{
		Password: &server.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetToken())

	return resp.GetToken()
}

// findGroup returns the group with the given name from the list, or nil.
func findGroup(groups []*sourcemodv1.Group, name string) *sourcemodv1.Group {
	for _, group := range groups {
		if group.GetName() == name {
			return group
		}
	}

	return nil
}

func TestSourcemodServiceAuth(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	missingImmunityID := int32(1)
	_, err := harness.anonWebClient().DeleteImmunity(ctx, &sourcemodv1.DeleteImmunityRequest{ImmunityId: &missingImmunityID})
	requireCode(t, connect.CodeUnauthenticated, err)

	_, err = harness.anonWebClient().Groups(ctx, &emptypb.Empty{})
	requireCode(t, connect.CodeUnauthenticated, err)

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)

	// The middleware maps a permission denial to an unauthenticated error.
	_, err = harness.webClient(t, person).DeleteImmunity(ctx, &sourcemodv1.DeleteImmunityRequest{ImmunityId: &missingImmunityID})
	requireCode(t, connect.CodeUnauthenticated, err)

	_, err = harness.webClient(t, person).Groups(ctx, &emptypb.Empty{})
	requireCode(t, connect.CodeUnauthenticated, err)

	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ")

	groups, err := harness.webClient(t, person).Groups(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Empty(t, groups.GetGroups())

	// Read permission alone must not allow write operations.
	_, err = harness.webClient(t, person).DeleteImmunity(ctx, &sourcemodv1.DeleteImmunityRequest{ImmunityId: &missingImmunityID})
	requireCode(t, connect.CodeUnauthenticated, err)

	_, err = harness.webClient(t, person).CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-auth")})
	requireCode(t, connect.CodeUnauthenticated, err)

	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_WRITE")

	// A missing immunity id reaches the handler now, proving the route is authorised.
	_, err = harness.webClient(t, person).DeleteImmunity(ctx, &sourcemodv1.DeleteImmunityRequest{ImmunityId: &missingImmunityID})
	requireCode(t, connect.CodeInternal, err)

	created, err := harness.webClient(t, person).CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{
		Name:        new("svc-auth"),
		Permissions: []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_KICK},
	})
	require.NoError(t, err)
	require.Equal(t, "svc-auth", created.GetGroup().GetName())

	groupID := created.GetGroup().GetGroupId()
	_, err = harness.webClient(t, person).DeleteGroup(ctx, &sourcemodv1.DeleteGroupRequest{GroupId: &groupID})
	require.NoError(t, err)
}

func TestSourcemodServiceGroupCRUD(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)
	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ", "PERMISSION_GAMEADMIN_WRITE")
	client := harness.webClient(t, person)

	kickRcon := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_RCON,
	}

	created, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-group"), Permissions: kickRcon})
	require.NoError(t, err)

	group := created.GetGroup()
	require.NotZero(t, group.GetGroupId())
	require.Equal(t, "svc-group", group.GetName())
	require.Equal(t, kickRcon, group.GetPermissions())
	require.Zero(t, group.GetImmunityLevel())

	rootGroup, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{
		Name:        new("svc-root-group"),
		Permissions: []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT},
	})
	require.NoError(t, err)
	require.Equal(t, int32(100), rootGroup.GetGroup().GetImmunityLevel())

	rootID := rootGroup.GetGroup().GetGroupId()
	_, err = client.DeleteGroup(ctx, &sourcemodv1.DeleteGroupRequest{GroupId: &rootID})
	require.NoError(t, err)

	list, err := client.Groups(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	found := findGroup(list.GetGroups(), "svc-group")
	require.NotNil(t, found)
	require.Equal(t, kickRcon, found.GetPermissions())

	chat := []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CHAT}
	groupID := group.GetGroupId()
	edited, err := client.EditGroups(ctx, &sourcemodv1.EditGroupsRequest{
		GroupId:     &groupID,
		Name:        new("svc-group-renamed"),
		Permissions: chat,
	})
	require.NoError(t, err)
	require.Equal(t, "svc-group-renamed", edited.GetGroup().GetName())
	require.Equal(t, chat, edited.GetGroup().GetPermissions())

	missingID := int32(999999)
	_, err = client.EditGroups(ctx, &sourcemodv1.EditGroupsRequest{GroupId: &missingID, Name: new("svc-missing")})
	requireCode(t, connect.CodeNotFound, err)

	_, err = client.DeleteGroup(ctx, &sourcemodv1.DeleteGroupRequest{GroupId: &groupID})
	require.NoError(t, err)

	list, err = client.Groups(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Nil(t, findGroup(list.GetGroups(), "svc-group-renamed"))

	_, err = client.DeleteGroup(ctx, &sourcemodv1.DeleteGroupRequest{GroupId: &groupID})
	requireCode(t, connect.CodeNotFound, err)
}

func TestSourcemodServiceGroupOverrideCRUD(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)
	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ", "PERMISSION_GAMEADMIN_WRITE")
	client := harness.webClient(t, person)

	parent, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-ovr-parent")})
	require.NoError(t, err)
	groupID := parent.GetGroup().GetGroupId()
	t.Cleanup(func() {
		_, _ = client.DeleteGroup(context.Background(), &sourcemodv1.DeleteGroupRequest{GroupId: &groupID})
	})

	groupType := sourcemodv1.OverrideType_OVERRIDE_TYPE_GROUP
	deny := sourcemodv1.OverrideAccess_OVERRIDE_ACCESS_DENY
	created, err := client.CreateGroupOverride(ctx, &sourcemodv1.CreateGroupOverrideRequest{
		GroupId: &groupID,
		Type:    &groupType,
		Name:    new("smakick"),
		Access:  &deny,
	})
	require.NoError(t, err)

	createdOverride := created.GetGroupOverride()
	require.NotZero(t, createdOverride.GetGroupOverrideId())
	require.Equal(t, groupID, createdOverride.GetGroupId())
	require.Equal(t, "smakick", createdOverride.GetName())
	require.Equal(t, sourcemodv1.OverrideAccess_OVERRIDE_ACCESS_DENY, createdOverride.GetOverrideAccess())

	list, err := client.GroupOverrides(ctx, &sourcemodv1.GroupOverridesRequest{GroupId: &groupID})
	require.NoError(t, err)
	require.Len(t, list.GetOverrides(), 1)
	require.Equal(t, "smakick", list.GetOverrides()[0].GetName())

	overrideID := createdOverride.GetGroupOverrideId()
	commandType := sourcemodv1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED
	allow := sourcemodv1.OverrideAccess_OVERRIDE_ACCESS_ALLOW_UNSPECIFIED
	edited, err := client.EditGroupOverride(ctx, &sourcemodv1.EditGroupOverrideRequest{
		GroupId:         &groupID,
		GroupOverrideId: &overrideID,
		Name:            new("smakick"),
		OverrideType:    &commandType,
		OverrideAccess:  &allow,
	})
	require.NoError(t, err)
	require.Equal(t, sourcemodv1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED, edited.GetGroupOverride().GetOverrideType())
	require.Equal(t, sourcemodv1.OverrideAccess_OVERRIDE_ACCESS_ALLOW_UNSPECIFIED, edited.GetGroupOverride().GetOverrideAccess())

	missingID := int32(999999)
	_, err = client.EditGroupOverride(ctx, &sourcemodv1.EditGroupOverrideRequest{
		GroupId:         &groupID,
		GroupOverrideId: &missingID,
		Name:            new("svc-missing"),
		OverrideType:    &commandType,
		OverrideAccess:  &allow,
	})
	requireCode(t, connect.CodeNotFound, err)

	_, err = client.DeleteGroupOverride(ctx, &sourcemodv1.DeleteGroupOverrideRequest{GroupOverrideId: &overrideID})
	require.NoError(t, err)

	list, err = client.GroupOverrides(ctx, &sourcemodv1.GroupOverridesRequest{GroupId: &groupID})
	require.NoError(t, err)
	require.Empty(t, list.GetOverrides())
}

func TestSourcemodServiceAdminCRUD(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)
	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ", "PERMISSION_GAMEADMIN_WRITE")
	client := harness.webClient(t, person)

	adminSid := steamid.RandSID64()

	kickBan := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
	}

	steamAuth := sourcemodv1.AuthType_AUTH_TYPE_STEAM_UNSPECIFIED
	created, err := client.CreateAdmin(ctx, &sourcemodv1.CreateAdminRequest{
		AuthType:    &steamAuth,
		Identity:    new(adminSid.String()),
		Name:        new("svc-admin"),
		Permissions: kickBan,
	})
	require.NoError(t, err)

	admin := created.GetAdmin()
	require.NotZero(t, admin.GetAdminId())
	require.Equal(t, string(adminSid.Steam3()), admin.GetIdentity())
	require.Equal(t, "svc-admin", admin.GetName())
	require.Equal(t, "steam", admin.GetAuthType())
	require.Equal(t, kickBan, admin.GetPermissions())
	require.Len(t, admin.GetGroups(), 1)

	list, err := client.Admins(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	var found *sourcemodv1.Admin
	for _, candidate := range list.GetAdmins() {
		if candidate.GetAdminId() == admin.GetAdminId() {
			found = candidate
		}
	}
	require.NotNil(t, found)
	require.Equal(t, kickBan, found.GetPermissions())

	group, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-admin-group")})
	require.NoError(t, err)
	groupID := group.GetGroup().GetGroupId()
	t.Cleanup(func() {
		_, _ = client.DeleteGroup(context.Background(), &sourcemodv1.DeleteGroupRequest{GroupId: &groupID})
	})

	adminID := admin.GetAdminId()
	added, err := client.AddAdminGroup(ctx, &sourcemodv1.AddAdminGroupRequest{
		AdminId: &adminID,
		GroupId: &groupID,
	})
	require.NoError(t, err)
	require.Len(t, added.GetAdmin().GetGroups(), 2)

	banOnly := []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_BAN}
	identity := admin.GetIdentity()
	edited, err := client.EditAdmin(ctx, &sourcemodv1.EditAdminRequest{
		AdminId:     &adminID,
		AuthType:    &steamAuth,
		Identity:    &identity,
		Name:        new("svc-admin-edited"),
		Permissions: banOnly,
	})
	require.NoError(t, err)
	require.Equal(t, "svc-admin-edited", edited.GetAdmin().GetName())
	require.Equal(t, banOnly, edited.GetAdmin().GetPermissions())

	missingID := int64(999999)
	missingIdentity := "76561198000000000"
	_, err = client.EditAdmin(ctx, &sourcemodv1.EditAdminRequest{
		AdminId:  &missingID,
		AuthType: &steamAuth,
		Identity: &missingIdentity,
		Name:     new("svc-missing"),
	})
	requireCode(t, connect.CodeNotFound, err)

	_, err = client.DeleteAdminGroup(ctx, &sourcemodv1.DeleteAdminGroupRequest{
		AdminId: &adminID,
		GroupId: &groupID,
	})
	require.NoError(t, err)

	_, err = client.DeleteAdmin(ctx, &sourcemodv1.DeleteAdminRequest{AdminId: &adminID})
	require.NoError(t, err)

	list, err = client.Admins(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, candidate := range list.GetAdmins() {
		require.NotEqual(t, adminID, candidate.GetAdminId())
	}
}

func TestSourcemodServiceOverrideCRUD(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)
	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ", "PERMISSION_GAMEADMIN_WRITE")
	client := harness.webClient(t, person)

	kick := []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_KICK}
	commandType := sourcemodv1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED

	created, err := client.CreateOverrides(ctx, &sourcemodv1.CreateOverridesRequest{
		Name:         new("svc-override"),
		OverrideType: &commandType,
		Permissions:  kick,
	})
	require.NoError(t, err)

	createdOverride := created.GetOverride()
	require.NotZero(t, createdOverride.GetOverrideId())
	require.Equal(t, "svc-override", createdOverride.GetName())
	require.Equal(t, sourcemodv1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED, createdOverride.GetOverrideType())
	require.Equal(t, kick, createdOverride.GetPermissions())

	list, err := client.Overrides(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	var found *sourcemodv1.Override
	for _, candidate := range list.GetOverrides() {
		if candidate.GetOverrideId() == createdOverride.GetOverrideId() {
			found = candidate
		}
	}
	require.NotNil(t, found)
	require.Equal(t, kick, found.GetPermissions())

	overrideID := createdOverride.GetOverrideId()
	slay := []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_SLAY}
	edited, err := client.EditOverrides(ctx, &sourcemodv1.EditOverridesRequest{
		OverrideId:   &overrideID,
		Name:         new("svc-override-renamed"),
		OverrideType: &commandType,
		Permissions:  slay,
	})
	require.NoError(t, err)
	require.Equal(t, "svc-override-renamed", edited.GetOverride().GetName())
	require.Equal(t, slay, edited.GetOverride().GetPermissions())

	missingID := int32(999999)
	_, err = client.EditOverrides(ctx, &sourcemodv1.EditOverridesRequest{
		OverrideId:   &missingID,
		Name:         new("svc-missing"),
		OverrideType: &commandType,
	})
	requireCode(t, connect.CodeNotFound, err)

	_, err = client.DeleteOverrides(ctx, &sourcemodv1.DeleteOverridesRequest{OverrideId: &overrideID})
	require.NoError(t, err)

	list, err = client.Overrides(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, candidate := range list.GetOverrides() {
		require.NotEqual(t, overrideID, candidate.GetOverrideId())
	}
}

func TestSourcemodServiceImmunities(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	person := fixture.CreateTestPerson(ctx, sid)
	harness.grantRole(t, sid, "PERMISSION_GAMEADMIN_READ", "PERMISSION_GAMEADMIN_WRITE")
	client := harness.webClient(t, person)

	groupA, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-imm-a")})
	require.NoError(t, err)
	groupB, err := client.CreateGroup(ctx, &sourcemodv1.CreateGroupRequest{Name: new("svc-imm-b")})
	require.NoError(t, err)

	idA := groupA.GetGroup().GetGroupId()
	idB := groupB.GetGroup().GetGroupId()
	t.Cleanup(func() {
		_, _ = client.DeleteGroup(context.Background(), &sourcemodv1.DeleteGroupRequest{GroupId: &idA})
		_, _ = client.DeleteGroup(context.Background(), &sourcemodv1.DeleteGroupRequest{GroupId: &idB})
	})

	created, err := client.CreateImmunity(ctx, &sourcemodv1.CreateImmunityRequest{GroupId: &idA, OtherId: &idB})
	require.NoError(t, err)

	immunity := created.GetGroupImmunity()
	require.NotZero(t, immunity.GetGroupImmunityId())
	require.Equal(t, "svc-imm-a", immunity.GetGroup().GetName())
	require.Equal(t, "svc-imm-b", immunity.GetOther().GetName())

	list, err := client.GroupImmunities(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, list.GetGroupImmunities(), 1)
	require.Equal(t, immunity.GetGroupImmunityId(), list.GetGroupImmunities()[0].GetGroupImmunityId())

	immunityID := immunity.GetGroupImmunityId()
	_, err = client.DeleteImmunity(ctx, &sourcemodv1.DeleteImmunityRequest{ImmunityId: &immunityID})
	require.NoError(t, err)

	list, err = client.GroupImmunities(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Empty(t, list.GetGroupImmunities())
}

func TestPluginServiceAuthenticate(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	server := fixture.CreateTestServer(ctx)

	_, err := harness.anonPluginClient().SMAuthenticate(ctx, &sourcemodv1.SMAuthenticateRequest{})
	requireCode(t, connect.CodePermissionDenied, err)

	wrongPassword := "not-the-password"
	_, err = harness.anonPluginClient().SMAuthenticate(ctx, &sourcemodv1.SMAuthenticateRequest{Password: &wrongPassword})
	requireCode(t, connect.CodePermissionDenied, err)

	resp, err := harness.anonPluginClient().SMAuthenticate(ctx, &sourcemodv1.SMAuthenticateRequest{Password: &server.Password})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetToken())
}

func TestPluginServiceServerAuth(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	_, err := harness.anonPluginClient().SMGroups(ctx, &emptypb.Empty{})
	requireCode(t, connect.CodeUnauthenticated, err)

	_, err = harness.pluginClient(t, "not-a-valid-token").SMGroups(ctx, &emptypb.Empty{})
	requireCode(t, connect.CodeUnauthenticated, err)

	resp, err := harness.pluginClient(t, harness.serverToken(t)).SMGroups(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestPluginServiceSMGroups(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	kickRcon := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_RCON,
	}

	group, err := harness.sm.AddGroup(ctx, "svc-sm-groups", kickRcon)
	require.NoError(t, err)

	root, err := harness.sm.AddGroup(ctx, "svc-sm-root", []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT})
	require.NoError(t, err)

	_, err = harness.sm.AddGroupImmunity(ctx, group.GroupID, root.GroupID)
	require.NoError(t, err)

	resp, err := harness.pluginClient(t, harness.serverToken(t)).SMGroups(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	found := findGroup(resp.GetGroups(), "svc-sm-groups")
	require.NotNil(t, found)
	require.Equal(t, kickRcon, found.GetPermissions())
	require.Zero(t, found.GetImmunityLevel())

	rootGroup := findGroup(resp.GetGroups(), "svc-sm-root")
	require.NotNil(t, rootGroup)
	require.Equal(t, int32(100), rootGroup.GetImmunityLevel())

	require.Len(t, resp.GetImmunities(), 1)
	require.Equal(t, "svc-sm-groups", resp.GetImmunities()[0].GetGroupName())
	require.Equal(t, "svc-sm-root", resp.GetImmunities()[0].GetOtherName())
}

func TestPluginServiceSMUsers(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	adminSid := steamid.RandSID64()

	kickBan := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
	}

	admin, err := harness.sm.AddAdmin(ctx, "svc-sm-admin", sourcemod.AuthTypeSteam, adminSid.String(), kickBan)
	require.NoError(t, err)
	require.Equal(t, kickBan, admin.Permissions)

	resp, err := harness.pluginClient(t, harness.serverToken(t)).SMUsers(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	var found *sourcemodv1.SMUser
	for _, user := range resp.GetUsers() {
		if user.GetIdentity() == string(adminSid.Steam3()) {
			found = user
		}
	}
	require.NotNil(t, found)
	require.Equal(t, admin.AdminID, found.GetId())
	require.Equal(t, "steam", found.GetAuthType())
	require.Equal(t, "svc-sm-admin", found.GetName())
	require.Equal(t, kickBan, found.GetPermissions())

	var groupLinks int
	for _, link := range resp.GetUserGroups() {
		if link.GetAdminId() == admin.AdminID {
			groupLinks++
		}
	}
	require.GreaterOrEqual(t, groupLinks, 1)
}

func TestPluginServiceSMOverrides(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	kickSlay := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_SLAY,
	}

	_, err := harness.sm.AddOverride(ctx, "svc-sm-override", sourcemod.OverrideTypeCommand, kickSlay)
	require.NoError(t, err)

	resp, err := harness.pluginClient(t, harness.serverToken(t)).SMOverrides(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	var found *sourcemodv1.SMOverride
	for _, override := range resp.GetOverrides() {
		if override.GetName() == "svc-sm-override" {
			found = override
		}
	}
	require.NotNil(t, found)
	require.Equal(t, sourcemodv1.OverrideType_OVERRIDE_TYPE_COMMAND_UNSPECIFIED, found.GetOverrideType())
	require.Equal(t, kickSlay, found.GetPermissions())
}

// smCheck runs an SMCheck for the given player and ip.
func (harness *rpcHarness) smCheck(t *testing.T, token, steamID, ipAddr string, clientID int32) (*sourcemodv1.SMCheckResponse, error) {
	t.Helper()

	name := "svc-player"

	return harness.pluginClient(t, token).SMCheck(t.Context(), &sourcemodv1.SMCheckRequest{
		SteamId:  new(steamID),
		ClientId: new(clientID),
		Ip:       new(ipAddr),
		Name:     &name,
	})
}

// createBan creates a ban via the ban service.
func (harness *rpcHarness) createBan(t *testing.T, opts ban.Opts) ban.Ban {
	t.Helper()

	created, err := harness.bans.Create(t.Context(), opts)
	require.NoError(t, err)

	return created
}

func TestPluginServiceSMCheck(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()
	token := harness.serverToken(t)
	clientID := int32(7)

	sid := steamid.RandSID64()
	fixture.CreateTestPerson(ctx, sid)

	t.Run("unbanned player connects", func(t *testing.T) {
		resp, err := harness.smCheck(t, token, sid.String(), "192.168.1.100", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_OK_UNSPECIFIED, resp.GetBanType())
		require.Equal(t, clientID, resp.GetClientId())
	})

	t.Run("fails open on invalid ip", func(t *testing.T) {
		resp, err := harness.smCheck(t, token, sid.String(), "not-an-ip", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_OK_UNSPECIFIED, resp.GetBanType())
	})

	t.Run("banned player is denied", func(t *testing.T) {
		harness.createBan(t, ban.Opts{
			SourceID:   tests.OwnerSID,
			TargetID:   sid,
			ValidUntil: time.Now().Add(time.Hour),
			BanType:    bantype.Banned,
			Reason:     reason.Cheating,
			Origin:     ban.Web,
		})

		resp, err := harness.smCheck(t, token, sid.String(), "192.168.1.101", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_BANNED, resp.GetBanType())
		require.Contains(t, resp.GetMsg(), "Banned")
	})

	t.Run("muted player is denied", func(t *testing.T) {
		mutedSid := steamid.RandSID64()
		fixture.CreateTestPerson(ctx, mutedSid)

		harness.createBan(t, ban.Opts{
			SourceID:   tests.OwnerSID,
			TargetID:   mutedSid,
			ValidUntil: time.Now().Add(time.Hour),
			BanType:    bantype.NoComm,
			Reason:     reason.Spam,
			Origin:     ban.Web,
		})

		resp, err := harness.smCheck(t, token, mutedSid.String(), "192.168.1.102", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_NO_COMM, resp.GetBanType())
		require.Contains(t, resp.GetMsg(), "muted")
	})

	t.Run("network ban is denied", func(t *testing.T) {
		networkSid := steamid.RandSID64()
		fixture.CreateTestPerson(ctx, networkSid)

		cidr := "198.51.100.0/28"
		harness.createBan(t, ban.Opts{
			SourceID:   tests.OwnerSID,
			TargetID:   networkSid,
			ValidUntil: time.Now().Add(time.Hour),
			BanType:    bantype.Network,
			Reason:     reason.External,
			Origin:     ban.Web,
			CIDR:       &cidr,
		})

		resp, err := harness.smCheck(t, token, networkSid.String(), "198.51.100.10", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_NETWORK, resp.GetBanType())
	})

	t.Run("evasion ban for banned network", func(t *testing.T) {
		bannedSid := steamid.RandSID64()
		evasiveSid := steamid.RandSID64()
		for _, s := range []steamid.SteamID{bannedSid, evasiveSid} {
			fixture.CreateTestPerson(ctx, s)
		}

		cidr := "203.0.113.0/28"
		harness.createBan(t, ban.Opts{
			SourceID:   tests.OwnerSID,
			TargetID:   bannedSid,
			ValidUntil: time.Now().Add(time.Hour),
			BanType:    bantype.Banned,
			Reason:     reason.Cheating,
			Origin:     ban.Web,
			CIDR:       &cidr,
		})

		// A different player connecting from within the banned range gets an
		// evasion ban via the ban service.
		resp, err := harness.smCheck(t, token, evasiveSid.String(), "203.0.113.5", clientID)
		require.NoError(t, err)
		require.Equal(t, banv1.BanType_BAN_TYPE_BANNED, resp.GetBanType())
		require.Equal(t, "Evasion ban", resp.GetMsg())

		evadeBan, err := harness.bans.QueryOne(ctx, ban.QueryOpts{
			TargetID: evasiveSid,
			Reasons:  []reason.Reason{reason.Evading},
		})
		require.NoError(t, err)
		require.Equal(t, bantype.Banned, evadeBan.BanType)
		require.NotZero(t, evadeBan.BanID)
	})
}

func TestPluginServiceSMPingMod(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()
	clientID := int32(3)

	req := &sourcemodv1.SMPingModRequest{
		SteamId:  new(sid.String()),
		Name:     new("svc-pingmod"),
		Reason:   new("suspected cheater"),
		ClientId: &clientID,
	}

	_, err := harness.anonPluginClient().SMPingMod(ctx, req)
	requireCode(t, connect.CodeUnauthenticated, err)

	token := harness.serverToken(t)

	invalidSID := "not-a-steamid"
	_, err = harness.pluginClient(t, token).SMPingMod(ctx, &sourcemodv1.SMPingModRequest{SteamId: &invalidSID})
	requireCode(t, connect.CodeInvalidArgument, err)

	_, err = harness.pluginClient(t, token).SMPingMod(ctx, req)
	require.NoError(t, err)

	_, err = harness.pluginClient(t, token).SMPingMod(ctx, req)
	requireCode(t, connect.CodeResourceExhausted, err)
}

func TestPluginServiceSMSeed(t *testing.T) {
	harness := newRPCHarness(t)
	ctx := t.Context()

	sid := steamid.RandSID64()

	_, err := harness.anonPluginClient().SMSeed(ctx, &sourcemodv1.SMSeedRequest{SteamId: new(sid.String())})
	requireCode(t, connect.CodeUnauthenticated, err)

	seedToken := harness.seedServerToken(t)

	invalidSID := "not-a-steamid"
	_, err = harness.pluginClient(t, seedToken).SMSeed(ctx, &sourcemodv1.SMSeedRequest{SteamId: &invalidSID})
	requireCode(t, connect.CodeInvalidArgument, err)

	resp, err := harness.pluginClient(t, seedToken).SMSeed(ctx, &sourcemodv1.SMSeedRequest{SteamId: new(sid.String())})
	require.NoError(t, err)
	require.Equal(t, "Successfully sent request", resp.GetMessage())

	// The seed queue rate limits the same user across all servers.
	_, err = harness.pluginClient(t, seedToken).SMSeed(ctx, &sourcemodv1.SMSeedRequest{SteamId: new(sid.String())})
	requireCode(t, connect.CodeResourceExhausted, err)
}
