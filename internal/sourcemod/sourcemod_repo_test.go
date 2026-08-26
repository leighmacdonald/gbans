package sourcemod_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
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

func createTestAdmin(t *testing.T, repo sourcemod.Repository, sid steamid.SteamID, name, flags string) sourcemod.Admin {
	t.Helper()
	ctx := t.Context()

	fixture.CreateTestPerson(ctx, sid)

	admin, err := repo.AddAdmin(ctx, sourcemod.Admin{
		AdminID:   sid.Int64(),
		SteamID:   sid,
		AuthType:  sourcemod.AuthTypeSteam,
		Identity:  string(sid.Steam3()),
		Password:  "",
		Flags:     flags,
		Name:      name,
		Immunity:  0,
		Groups:    []sourcemod.Groups{},
		CreatedOn: time.Time{},
		UpdatedOn: time.Time{},
	})
	require.NoError(t, err)

	return admin
}

func createTestGroup(t *testing.T, repo sourcemod.Repository, name, flags string) sourcemod.Groups {
	t.Helper()

	group, err := repo.AddGroup(t.Context(), sourcemod.Groups{
		GroupID:       0,
		Flags:         flags,
		Name:          name,
		ImmunityLevel: 0,
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
	admin := createTestAdmin(t, repo, sid, "smoke-admin", "bl")

	require.Equal(t, sid.Int64(), admin.AdminID)
	require.Equal(t, "smoke-admin", admin.Name)
	require.Equal(t, "bl", admin.Flags)
	require.Equal(t, int32(0), admin.Immunity)
	require.Len(t, admin.Groups, 1)
	require.Equal(t, "smoke-admin", admin.Groups[0].Name)
	require.Equal(t, "bl", admin.Groups[0].Flags)

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

	t.Run("root flag derives immunity", func(t *testing.T) {
		rootSID := steamid.New(76561198000777002)
		root := createTestAdmin(t, repo, rootSID, "smoke-root", "bz")

		require.Equal(t, "zb", root.Flags)
		require.Equal(t, int32(100), root.Immunity)

		require.NoError(t, repo.DelAdmin(ctx, root))
		_, err := repo.GetAdminByID(ctx, rootSID.Int64())
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("save renames the personal role and updates flags", func(t *testing.T) {
		saved, err := repo.SaveAdmin(ctx, sourcemod.Admin{
			AdminID:   sid.Int64(),
			SteamID:   sid,
			AuthType:  sourcemod.AuthTypeSteam,
			Identity:  string(sid.Steam3()),
			Password:  "",
			Flags:     "bk",
			Name:      "smoke-renamed-admin",
			Immunity:  0,
			Groups:    []sourcemod.Groups{},
			CreatedOn: admin.CreatedOn,
			UpdatedOn: admin.UpdatedOn,
		})
		require.NoError(t, err)
		require.Equal(t, "smoke-renamed-admin", saved.Name)
		require.Equal(t, "bk", saved.Flags)

		missing, err := repo.GetGroupByName(ctx, "smoke-admin")
		require.ErrorIs(t, err, database.ErrNoResult)
		require.Zero(t, missing.GroupID)

		renamed, err := repo.GetGroupByName(ctx, "smoke-renamed-admin")
		require.NoError(t, err)
		require.Equal(t, "bk", renamed.Flags)
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
	admin := createTestAdmin(t, repo, sid, "smoke-member", "bl")
	group := createTestGroup(t, repo, "smoke-testers", "bd")

	byName, err := repo.GetGroupByName(ctx, "smoke-testers")
	require.NoError(t, err)
	require.Equal(t, group.GroupID, byName.GroupID)

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
			Flags:         "bd",
			Name:          "smoke-renamed",
			ImmunityLevel: 0,
			CreatedOn:     group.CreatedOn,
			UpdatedOn:     time.Time{},
		})
		require.NoError(t, err)
		require.Equal(t, "smoke-renamed", saved.Name)

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

	groupA := createTestGroup(t, repo, "smoke-immunity-a", "b")
	groupB := createTestGroup(t, repo, "smoke-immunity-b", "b")

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

	group := createTestGroup(t, repo, "smoke-overrides", "b")

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

	override, err := repo.AddOverride(ctx, sourcemod.Overrides{
		OverrideID: 0,
		Type:       sourcemod.OverrideTypeCommand,
		Name:       "smoke-custom",
		Flags:      "a",
		CreatedOn:  time.Time{},
		UpdatedOn:  time.Time{},
	})
	require.NoError(t, err)
	require.NotZero(t, override.OverrideID)

	overrides, err := repo.Overrides(ctx)
	require.NoError(t, err)
	require.Len(t, overrides, 1)
	require.Equal(t, override.OverrideID, overrides[0].OverrideID)
	require.Equal(t, "smoke-custom", overrides[0].Name)
	require.Equal(t, "a", overrides[0].Flags)

	updated, err := repo.SaveOverride(ctx, sourcemod.Overrides{
		OverrideID: override.OverrideID,
		Type:       sourcemod.OverrideTypeCommand,
		Name:       "smoke-custom",
		Flags:      "b",
		CreatedOn:  override.CreatedOn,
		UpdatedOn:  time.Time{},
	})
	require.NoError(t, err)

	byID, err := repo.GetOverride(ctx, updated.OverrideID)
	require.NoError(t, err)
	require.Equal(t, "b", byID.Flags)

	require.NoError(t, repo.DelOverride(ctx, byID))

	remaining, err := repo.Overrides(ctx)
	require.NoError(t, err)
	require.NotContains(t, remaining, byID)
}
