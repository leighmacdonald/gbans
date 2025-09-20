package tests_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/stretchr/testify/require"
)

func TestPerson(t *testing.T) {
	router := testRouter()
	source := getUser()
	sourceAuth := loginUser(source)
	modAuth := loginUser(getModerator())
	adminAuth := loginUser(getOwner())

	var prof person.ProfileResponse
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/profile", httphelper.RequestQuery{Query: source.SteamID.String()}, http.StatusOK, nil, &prof)
	require.Equal(t, source.SteamID, prof.Player.SteamID)
	require.NotEmpty(t, prof.Player.AvatarHash)

	var profile domain.PersonCore
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/current_profile", nil, http.StatusOK, &authTokens{user: sourceAuth}, &profile)
	require.Equal(t, source.SteamID, profile.SteamID)

	var settings person.Settings
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/current_profile/settings", nil, http.StatusOK, &authTokens{user: sourceAuth}, &settings)

	var updated person.Settings
	update := person.SettingsUpdate{
		ForumSignature:       settings.ForumSignature + "x",
		ForumProfileMessages: !settings.ForumProfileMessages,
		StatsHidden:          !settings.StatsHidden,
	}
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/current_profile/settings", update, http.StatusOK, &authTokens{user: sourceAuth}, &updated)

	require.Equal(t, update.ForumSignature, updated.ForumSignature)
	require.Equal(t, update.StatsHidden, updated.StatsHidden)
	require.Equal(t, update.ForumProfileMessages, updated.ForumProfileMessages)

	var res httphelper.LazyResult
	query := person.PlayerQuery{
		TargetIDField: domain.TargetIDField{TargetID: source.SteamID.String()},
	}
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/players", query, http.StatusOK, &authTokens{user: modAuth}, &res)
	require.Len(t, res.Data, 1)

	newPerms := person.RequestPermissionLevelUpdate{PermissionLevel: permission.PModerator}
	var newMod person.Person
	testEndpointWithReceiver(t, router, http.MethodPut, fmt.Sprintf("/api/player/%s/permissions", profile.SteamID.String()), newPerms, http.StatusOK, &authTokens{user: adminAuth}, &newMod)
	require.Equal(t, newPerms.PermissionLevel, newMod.PermissionLevel)
}

func TestPersonPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/current_profile",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/current_profile/settings",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/current_profile/settings",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/players",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/player/1/permissions",
			method: http.MethodPut,
			code:   http.StatusForbidden,
			levels: admin,
		},
	})
}
