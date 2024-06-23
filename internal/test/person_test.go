package test_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPerson(t *testing.T) {
	router := testRouter()
	source := getUser()
	sourceAuth := loginUser(source)
	modAuth := loginUser(getModerator())
	adminAuth := loginUser(getOwner())

	var prof domain.ProfileResponse
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/profile", domain.RequestQuery{Query: source.SteamID.String()}, http.StatusOK, nil, &prof)
	require.Equal(t, source.SteamID, prof.Player.SteamID)
	require.NotEmpty(t, prof.Player.AvatarHash)

	var profile domain.UserProfile
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/current_profile", nil, http.StatusOK, sourceAuth, &profile)
	require.Equal(t, source.SteamID, profile.SteamID)

	var settings domain.PersonSettings
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/current_profile/settings", nil, http.StatusOK, sourceAuth, &settings)

	var updated domain.PersonSettings
	update := domain.PersonSettingsUpdate{
		ForumSignature:       settings.ForumSignature + "x",
		ForumProfileMessages: !settings.ForumProfileMessages,
		StatsHidden:          !settings.StatsHidden,
	}
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/current_profile/settings", update, http.StatusOK, sourceAuth, &updated)

	require.Equal(t, update.ForumSignature, updated.ForumSignature)
	require.Equal(t, update.StatsHidden, updated.StatsHidden)
	require.Equal(t, update.ForumProfileMessages, updated.ForumProfileMessages)

	var res domain.LazyResult
	query := domain.PlayerQuery{
		TargetIDField: domain.TargetIDField{TargetID: source.SteamID.String()},
	}
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/players", query, http.StatusOK, modAuth, &res)
	require.Len(t, res.Data, 1)

	newPerms := domain.RequestPermissionLevelUpdate{PermissionLevel: domain.PModerator}
	var newMod domain.Person
	testEndpointWithReceiver(t, router, http.MethodPut, fmt.Sprintf("/api/player/%s/permissions", profile.SteamID.String()), newPerms, http.StatusOK, adminAuth, &newMod)
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
