package sourcemod_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func createTestAdmin(t *testing.T, repo sourcemod.Repository, sid steamid.SteamID, name string, perms []rolesv1.Permission) sourcemod.Admin {
	t.Helper()
	ctx := t.Context()

	fixture.CreateTestPerson(ctx, sid)

	admin, err := repo.AddAdmin(ctx, sourcemod.Admin{
		AdminID:     sid.Int64(),
		SteamID:     sid,
		AuthType:    sourcemod.AuthTypeSteam,
		Identity:    string(sid.Steam3()),
		Password:    "",
		Name:        name,
		Immunity:    0,
		Groups:      []sourcemod.Groups{},
		Permissions: perms,
		CreatedOn:   time.Time{},
		UpdatedOn:   time.Time{},
	})
	require.NoError(t, err)

	return admin
}

func createTestGroup(t *testing.T, repo sourcemod.Repository, name string, perms []rolesv1.Permission) sourcemod.Groups {
	t.Helper()

	group, err := repo.AddGroup(t.Context(), sourcemod.Groups{
		GroupID:       0,
		Name:          name,
		ImmunityLevel: 0,
		Permissions:   perms,
		CreatedOn:     time.Time{},
		UpdatedOn:     time.Time{},
	})
	require.NoError(t, err)

	return group
}

func TestAdminLifecycle(t *testing.T) {
	repo := sourcemod.NewRepository(fixture.Database)
	ctx := t.Context()

	sid := steamid.New(76561198000777001)
	kickRcon := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_RCON,
	}
	admin := createTestAdmin(t, repo, sid, "smoke-admin", kickRcon)

	require.Equal(t, sid.Int64(), admin.AdminID)
	require.Equal(t, "smoke-admin", admin.Name)
	require.Equal(t, int32(0), admin.Immunity)
	require.Len(t, admin.Groups, 1)
	require.Equal(t, "smoke-admin", admin.Groups[0].Name)
	require.Equal(t, kickRcon, admin.Permissions)
	require.Equal(t, kickRcon, admin.Groups[0].Permissions)

	t.Run("reads back by id and lists admins", func(t *testing.T) {
		byID, err := repo.GetAdminByID(ctx, sid.Int64())
		require.NoError(t, err)
		require.Equal(t, admin, byID)

		admins, err := repo.Admins(ctx)
		require.NoError(t, err)
		require.Contains(t, admins, admin)

		missingSID := steamid.New(76561198000777099)
		missing, err := repo.GetAdminByID(ctx, missingSID.Int64())
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, missing.AdminID)
	})

	t.Run("root permission derives immunity", func(t *testing.T) {
		rootSID := steamid.New(76561198000777002)
		kickRoot := []rolesv1.Permission{
			rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
			rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT,
		}
		root := createTestAdmin(t, repo, rootSID, "smoke-root", kickRoot)

		require.Equal(t, int32(100), root.Immunity)
		require.Equal(t, kickRoot, root.Permissions)

		require.NoError(t, repo.DelAdmin(ctx, root))
		_, err := repo.GetAdminByID(ctx, rootSID.Int64())
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("save renames the personal role and updates permissions", func(t *testing.T) {
		kickVote := []rolesv1.Permission{
			rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
			rolesv1.Permission_PERMISSION_SOURCEMOD_VOTE,
		}
		saved, err := repo.SaveAdmin(ctx, sourcemod.Admin{
			AdminID:     sid.Int64(),
			SteamID:     sid,
			AuthType:    sourcemod.AuthTypeSteam,
			Identity:    string(sid.Steam3()),
			Password:    "",
			Name:        "smoke-renamed-admin",
			Immunity:    0,
			Groups:      []sourcemod.Groups{},
			Permissions: kickVote,
			CreatedOn:   admin.CreatedOn,
			UpdatedOn:   admin.UpdatedOn,
		})
		require.NoError(t, err)
		require.Equal(t, "smoke-renamed-admin", saved.Name)
		require.Equal(t, kickVote, saved.Permissions)

		missing, err := repo.GetGroupByName(ctx, "smoke-admin")
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, missing.GroupID)

		renamed, err := repo.GetGroupByName(ctx, "smoke-renamed-admin")
		require.NoError(t, err)
		require.Equal(t, kickVote, renamed.Permissions)
	})

	t.Run("delete admin removes the personal role", func(t *testing.T) {
		require.NoError(t, repo.DelAdmin(ctx, admin))

		missing, err := repo.GetAdminByID(ctx, sid.Int64())
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, missing.AdminID)

		role, err := repo.GetGroupByName(ctx, "smoke-renamed-admin")
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, role.GroupID)
	})
}

func TestAdminGroupAssignment(t *testing.T) {
	repo := sourcemod.NewRepository(fixture.Database)
	ctx := t.Context()

	sid := steamid.New(76561198000777003)
	kickRcon := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_RCON,
	}
	banUnban := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
		rolesv1.Permission_PERMISSION_SOURCEMOD_UNBAN,
	}
	admin := createTestAdmin(t, repo, sid, "smoke-member", kickRcon)
	group := createTestGroup(t, repo, "smoke-testers", banUnban)

	byName, err := repo.GetGroupByName(ctx, "smoke-testers")
	require.NoError(t, err)
	require.Equal(t, group.GroupID, byName.GroupID)
	require.Equal(t, banUnban, byName.Permissions)

	groups, err := repo.Groups(ctx)
	require.NoError(t, err)
	require.Contains(t, groups, byName)

	t.Run("assign and unassign", func(t *testing.T) {
		require.NoError(t, repo.InsertAdminGroup(ctx, admin, group))

		withGroup, err := repo.GetAdminByID(ctx, sid.Int64())
		require.NoError(t, err)
		require.Len(t, withGroup.Groups, 2)
		groupNames := make([]string, 0, len(withGroup.Groups))
		for _, group := range withGroup.Groups {
			groupNames = append(groupNames, group.Name)
		}
		require.ElementsMatch(t, []string{"smoke-member", "smoke-testers"}, groupNames)

		require.NoError(t, repo.DeleteAdminGroup(ctx, admin, group))

		withoutGroup, err := repo.GetAdminByID(ctx, sid.Int64())
		require.NoError(t, err)
		require.Len(t, withoutGroup.Groups, 1)
		require.Equal(t, "smoke-member", withoutGroup.Groups[0].Name)
	})

	t.Run("rename and delete group", func(t *testing.T) {
		saved, err := repo.SaveGroup(ctx, sourcemod.Groups{
			GroupID:       group.GroupID,
			Name:          "smoke-renamed",
			ImmunityLevel: 0,
			Permissions:   banUnban,
			CreatedOn:     group.CreatedOn,
			UpdatedOn:     time.Time{},
		})
		require.NoError(t, err)
		require.Equal(t, "smoke-renamed", saved.Name)
		require.Equal(t, byName.Permissions, saved.Permissions)

		missing, err := repo.GetGroupByName(ctx, "smoke-testers")
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, missing.GroupID)

		require.NoError(t, repo.DeleteGroup(ctx, saved))
		_, err = repo.GetGroupByID(ctx, group.GroupID)
		require.ErrorIs(t, err, database.ErrNoResult)

		require.NoError(t, repo.DelAdmin(ctx, admin))
	})
}

func TestGroupImmunities(t *testing.T) {
	repo := sourcemod.NewRepository(fixture.Database)
	ctx := t.Context()

	ban := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
	}
	groupA := createTestGroup(t, repo, "smoke-immunity-a", ban)
	groupB := createTestGroup(t, repo, "smoke-immunity-b", ban)

	immunity, err := repo.AddGroupImmunity(ctx, groupA, groupB)
	require.NoError(t, err)
	require.NotZero(t, immunity.GroupImmunityID)

	byID, err := repo.GetGroupImmunityByID(ctx, immunity.GroupImmunityID)
	require.NoError(t, err)
	require.Equal(t, groupA.GroupID, byID.Group.GroupID)
	require.Equal(t, groupB.GroupID, byID.Other.GroupID)

	immunities, err := repo.GetGroupImmunities(ctx)
	require.NoError(t, err)
	require.Contains(t, immunities, byID)

	require.NoError(t, repo.DelGroupImmunity(ctx, byID))

	missing, err := repo.GetGroupImmunityByID(ctx, immunity.GroupImmunityID)
	require.ErrorIs(t, err, database.ErrNoResult)
	require.Zero(t, missing.GroupImmunityID)

	require.NoError(t, repo.DeleteGroup(ctx, groupA))
	require.NoError(t, repo.DeleteGroup(ctx, groupB))
}

func TestGroupOverrides(t *testing.T) {
	repo := sourcemod.NewRepository(fixture.Database)
	ctx := t.Context()

	ban := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
	}
	group := createTestGroup(t, repo, "smoke-overrides", ban)

	override, err := repo.AddGroupOverride(ctx, sourcemod.GroupOverrides{
		GroupOverrideID: 0,
		GroupID:         group.GroupID,
		Type:            sourcemod.OverrideTypeCommand,
		Name:            "smakick",
		Access:          sourcemod.OverrideAccessDeny,
		CreatedOn:       time.Time{},
		UpdatedOn:       time.Time{},
	})
	require.NoError(t, err)
	require.NotZero(t, override.GroupOverrideID)

	overrides, err := repo.GroupOverrides(ctx, group)
	require.NoError(t, err)
	require.Len(t, overrides, 1)
	require.Equal(t, "smakick", overrides[0].Name)
	require.Equal(t, sourcemod.OverrideAccessDeny, overrides[0].Access)

	updated, err := repo.SaveGroupOverride(ctx, sourcemod.GroupOverrides{
		GroupOverrideID: override.GroupOverrideID,
		GroupID:         group.GroupID,
		Type:            sourcemod.OverrideTypeCommand,
		Name:            "smakick",
		Access:          sourcemod.OverrideAccessAllow,
		CreatedOn:       override.CreatedOn,
		UpdatedOn:       time.Time{},
	})
	require.NoError(t, err)

	byID, err := repo.GetGroupOverride(ctx, updated.GroupOverrideID)
	require.NoError(t, err)
	require.Equal(t, sourcemod.OverrideAccessAllow, byID.Access)

	require.NoError(t, repo.DelGroupOverride(ctx, byID))

	remaining, err := repo.GroupOverrides(ctx, group)
	require.NoError(t, err)
	require.Empty(t, remaining)

	require.NoError(t, repo.DeleteGroup(ctx, group))
}

func TestCommandOverrides(t *testing.T) {
	repo := sourcemod.NewRepository(fixture.Database)
	ctx := t.Context()

	kickRcon := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_RCON,
	}
	override, err := repo.AddOverride(ctx, sourcemod.Overrides{
		OverrideID:  0,
		Type:        sourcemod.OverrideTypeCommand,
		Name:        "smoke-custom",
		Permissions: kickRcon,
		CreatedOn:   time.Time{},
		UpdatedOn:   time.Time{},
	})
	require.NoError(t, err)
	require.NotZero(t, override.OverrideID)

	overrides, err := repo.Overrides(ctx)
	require.NoError(t, err)
	require.Len(t, overrides, 1)
	require.Equal(t, override.OverrideID, overrides[0].OverrideID)
	require.Equal(t, "smoke-custom", overrides[0].Name)
	require.Equal(t, kickRcon, overrides[0].Permissions)

	kickBan := []rolesv1.Permission{
		rolesv1.Permission_PERMISSION_SOURCEMOD_KICK,
		rolesv1.Permission_PERMISSION_SOURCEMOD_BAN,
	}
	updated, err := repo.SaveOverride(ctx, sourcemod.Overrides{
		OverrideID:  override.OverrideID,
		Type:        sourcemod.OverrideTypeCommand,
		Name:        "smoke-custom",
		Permissions: kickBan,
		CreatedOn:   override.CreatedOn,
		UpdatedOn:   time.Time{},
	})
	require.NoError(t, err)

	byID, err := repo.GetOverride(ctx, updated.OverrideID)
	require.NoError(t, err)
	require.Equal(t, kickBan, byID.Permissions)

	require.NoError(t, repo.DelOverride(ctx, byID))

	remaining, err := repo.Overrides(ctx)
	require.NoError(t, err)
	require.NotContains(t, remaining, byID)
}
