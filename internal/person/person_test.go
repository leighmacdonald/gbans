package person_test

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestPerson(t *testing.T) {
	personCase := person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi)

	_, err := personCase.BySteamID(t.Context(), steamid.RandSID64())
	require.Error(t, err)

	for _, sid := range []steamid.SteamID{tests.OwnerSID, tests.ModSID, tests.UserSID, tests.GuestSID} {
		user := person.New(sid)
		require.NoError(t, personCase.Save(t.Context(), &user))
	}

	fetched, errFetched := personCase.BySteamID(t.Context(), tests.UserSID)
	require.NoError(t, errFetched)

	fetched.PersonaName = stringutil.SecureRandomString(10)
	fetched.DiscordID = strconv.FormatInt(rand.Int64(), 10) //nolint:gosec

	require.NoError(t, personCase.Save(t.Context(), &fetched))

	defaultSettings, errSettings := personCase.GetPersonSettings(t.Context(), fetched.SteamID)
	require.NoError(t, errSettings)
	require.False(t, defaultSettings.MatchNotifications)

	updateValues := person.SettingsUpdate{
		StatsHidden:        true,
		MatchNotifications: true,
	}

	_, errUpdate := personCase.SavePersonSettings(t.Context(), fetched, updateValues)

	require.NoError(t, errUpdate)
	settings, errSettingsUpdate := personCase.GetPersonSettings(t.Context(), fetched.SteamID)
	require.NoError(t, errSettingsUpdate)

	require.Equal(t, updateValues.StatsHidden, settings.StatsHidden)
	require.Equal(t, updateValues.MatchNotifications, settings.MatchNotifications)

	players, _, errPlayers := personCase.GetPeople(t.Context(), person.Query{})
	require.NoError(t, errPlayers)
	require.GreaterOrEqual(t, len(players), 4)

	discord, errDisc := personCase.GetPersonByDiscordID(t.Context(), fetched.DiscordID)
	require.NoError(t, errDisc)
	require.Equal(t, fetched.SteamID, discord.SteamID)

	steamID, _ := personCase.BySteamID(t.Context(), fetched.SteamID)
	require.Equal(t, fetched.DiscordID, steamID.DiscordID)
}

func TestClientPrefs(t *testing.T) {
	const setupQuery = `
		INSERT INTO sm_cookies (name, description, access) VALUES
			('tfdb_speedhud_pref', 'Toggle for the Dodgeball Speed HUD', 0),
			('fov_override', 'FOV', 0),
			('no_motd', 'No MOTD', 1),
			('latest_class', 'Latest class', 2);`
	require.NoError(t, fixture.Database.Exec(t.Context(), setupQuery))

	sid := steamid.RandSID64()
	repo := person.NewRepository(fixture.Database, true)

	prefs, err := repo.ClientPrefs(t.Context(), sid)
	require.NoError(t, err)
	require.Len(t, prefs, 3)

	byName := make(map[string]person.ClientPref)
	for _, p := range prefs {
		byName[p.Name] = p
	}

	// Private cookies are never returned, protected ones are read-only.
	require.NotContains(t, byName, "latest_class")
	require.Equal(t, person.ClientPrefTypeBoolean, byName["tfdb_speedhud_pref"].Type)
	require.Equal(t, person.ClientPrefTypeNumber, byName["fov_override"].Type)
	require.True(t, byName["tfdb_speedhud_pref"].Editable())
	require.True(t, byName["fov_override"].Editable())
	require.False(t, byName["no_motd"].Editable())

	// Public values are persisted; protected and private names are ignored.
	require.NoError(t, repo.SaveClientPrefs(t.Context(), sid, map[string]string{
		"tfdb_speedhud_pref": "1",
		"fov_override":       "110",
		"no_motd":            "1",
		"latest_class":       "1",
	}))

	prefs, err = repo.ClientPrefs(t.Context(), sid)
	require.NoError(t, err)
	byName = make(map[string]person.ClientPref)
	for _, p := range prefs {
		byName[p.Name] = p
	}

	require.Equal(t, "1", byName["tfdb_speedhud_pref"].Value)
	require.Equal(t, "110", byName["fov_override"].Value)
	require.Empty(t, byName["no_motd"].Value)

	// An empty value removes the player's override.
	require.NoError(t, repo.SaveClientPrefs(t.Context(), sid, map[string]string{
		"fov_override": "",
	}))

	prefs, err = repo.ClientPrefs(t.Context(), sid)
	require.NoError(t, err)
	byName = make(map[string]person.ClientPref)
	for _, p := range prefs {
		byName[p.Name] = p
	}

	require.Empty(t, byName["fov_override"].Value)

	// Values are validated against the cookie's type.
	for _, invalid := range []string{"true", "2", "1.5"} {
		require.ErrorIs(t, repo.SaveClientPrefs(t.Context(), sid,
			map[string]string{"tfdb_speedhud_pref": invalid}), person.ErrClientPrefInvalidValue)
	}

	require.ErrorIs(t, repo.SaveClientPrefs(t.Context(), sid,
		map[string]string{"fov_override": "abc"}), person.ErrClientPrefInvalidValue)

	longValue := stringutil.SecureRandomString(101)
	require.ErrorIs(t, repo.SaveClientPrefs(t.Context(), sid,
		map[string]string{"fov_override": longValue}), person.ErrClientPrefValueTooLong)

	// Disabled repositories expose nothing and accept nothing.
	disabled := person.NewRepository(fixture.Database, false)
	disabledPrefs, err := disabled.ClientPrefs(t.Context(), sid)
	require.NoError(t, err)
	require.Empty(t, disabledPrefs)

	require.NoError(t, disabled.SaveClientPrefs(t.Context(), sid, map[string]string{
		"tfdb_speedhud_pref": "0",
	}))

	prefs, err = repo.ClientPrefs(t.Context(), sid)
	require.NoError(t, err)
	byName = make(map[string]person.ClientPref)
	for _, p := range prefs {
		byName[p.Name] = p
	}

	require.Equal(t, "1", byName["tfdb_speedhud_pref"].Value)
}

func TestRefreshFromSteam(t *testing.T) {
	personCase := person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi)

	sid := steamid.RandSID64()
	_, errCreate := personCase.GetOrCreatePersonBySteamID(t.Context(), sid)
	require.NoError(t, errCreate)

	stale := func(name string) {
		t.Helper()

		existing, errGet := personCase.BySteamID(t.Context(), sid)
		require.NoError(t, errGet)
		existing.PersonaName = name
		require.NoError(t, personCase.Save(t.Context(), &existing))
	}

	// A cached read must not touch a fresh profile.
	stale("stale-name")
	cached, errCached := personCase.GetOrCreatePersonBySteamID(t.Context(), sid)
	require.NoError(t, errCached)
	require.Equal(t, "stale-name", cached.Name)

	// The login path always pulls the current steam summary instead.
	stale("stale-name")
	refreshed, errRefresh := personCase.RefreshFromSteam(t.Context(), sid)
	require.NoError(t, errRefresh)
	require.Equal(t, "name-"+sid.String(), refreshed.Name)

	persisted, errPersisted := personCase.BySteamID(t.Context(), sid)
	require.NoError(t, errPersisted)
	require.Equal(t, "name-"+sid.String(), persisted.PersonaName)
	require.Equal(t, tests.DefaultAvatarHash, persisted.AvatarHash)
}

func TestActiveBanID(t *testing.T) {
	personCase := person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi)

	sid := steamid.RandSID64()
	created, errCreate := personCase.GetOrCreatePersonBySteamID(t.Context(), sid)
	require.NoError(t, errCreate)
	require.Zero(t, created.BanID)

	insertBan := func(validUntil time.Time) int32 {
		t.Helper()

		var banID int32
		require.NoError(t, fixture.Database.QueryRow(t.Context(),
			`INSERT INTO ban (target_id, ban_type, reason, valid_until, created_on, updated_on)
			 VALUES ($1, 2, 1, $2, now(), now()) RETURNING ban_id`,
			sid.Int64(), validUntil).Scan(&banID))

		return banID
	}

	banIDOf := func() int32 {
		t.Helper()

		fetched, errGet := personCase.BySteamID(t.Context(), sid)
		require.NoError(t, errGet)

		return fetched.BanID
	}

	// Expired bans are not active.
	expiredID := insertBan(time.Now().Add(-time.Hour))
	require.NotZero(t, expiredID)
	require.Zero(t, banIDOf())

	// Deleted (unbanned) bans are not active.
	deletedID := insertBan(time.Now().Add(time.Hour))
	require.NoError(t, fixture.Database.Exec(t.Context(),
		`UPDATE ban SET deleted = true WHERE ban_id = $1`, deletedID))
	require.Zero(t, banIDOf())

	// The newest active ban wins, including permanent ones.
	firstActiveID := insertBan(time.Now().Add(time.Hour))
	secondActiveID := insertBan(time.Now().AddDate(10, 0, 0))
	require.Greater(t, secondActiveID, firstActiveID)
	require.Equal(t, secondActiveID, banIDOf())

	refreshed, errRefresh := personCase.GetOrCreatePersonBySteamID(t.Context(), sid)
	require.NoError(t, errRefresh)
	require.Equal(t, secondActiveID, refreshed.BanID)
}
