package tests_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBans(t *testing.T) {
	router := testRouter()
	mod := getModerator()
	modCreds := loginUser(mod)
	target := getUser()

	// Ensure no bans exist
	var bansEmpty []domain.Ban
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/steam", nil, http.StatusOK, &authTokens{user: modCreds}, &bansEmpty)
	require.Len(t, bansEmpty, 1)

	// Create a ban
	banReq := domain.RequestBanCreate{
		SourceIDField:  domain.SourceIDField{SourceID: mod.SteamID.String()},
		TargetIDField:  domain.TargetIDField{TargetID: target.SteamID.String()},
		Duration:       "1d",
		BanType:        domain.Banned,
		Reason:         domain.Cheating,
		ReasonText:     "",
		Note:           "notes",
		ReportID:       0,
		DemoName:       "demo-test.dem",
		DemoTick:       100,
		IncludeFriends: true,
		EvadeOk:        true,
	}

	var fetchedBan domain.BannedPerson
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/bans/steam/create", banReq, http.StatusCreated, &authTokens{user: modCreds}, &fetchedBan)

	require.Equal(t, banReq.SourceID, fetchedBan.SourceID.String())
	require.Equal(t, banReq.TargetID, fetchedBan.TargetID.String())
	require.True(t, fetchedBan.ValidUntil.After(time.Now()))
	require.Equal(t, banReq.BanType, fetchedBan.BanType)
	require.Equal(t, banReq.Reason, fetchedBan.Reason)
	require.Equal(t, banReq.ReasonText, fetchedBan.ReasonText)
	require.Equal(t, banReq.Note, fetchedBan.Note)
	require.Equal(t, banReq.ReportID, fetchedBan.ReportID)
	require.Equal(t, banReq.IncludeFriends, fetchedBan.EvadeOk)
	require.Equal(t, banReq.IncludeFriends, fetchedBan.IncludeFriends)

	// Ensure it's in the ban collection
	var bans []domain.Ban
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/steam", nil, http.StatusOK, &authTokens{user: modCreds}, &bans)
	require.Len(t, bans, 2)

	updateReq := ban.RequestBanSteamUpdate{
		TargetID:       fetchedBan.TargetID,
		BanType:        domain.NoComm,
		Reason:         domain.Custom,
		ReasonText:     "blah",
		Note:           "edited",
		IncludeFriends: false,
		EvadeOk:        false,
		ValidUntil:     fetchedBan.ValidUntil.Add(time.Second * 10),
	}

	// Update the ban
	var updatedBan domain.BannedPerson
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/steam/%d", fetchedBan.BanID),
		updateReq, http.StatusOK, &authTokens{user: modCreds}, &updatedBan)

	require.Equal(t, updateReq.TargetID, updatedBan.TargetID)
	require.Equal(t, updateReq.BanType, updatedBan.BanType)
	require.Equal(t, updateReq.Reason, updatedBan.Reason)
	require.Equal(t, updateReq.ReasonText, updatedBan.ReasonText)
	require.Equal(t, updateReq.Note, updatedBan.Note)
	require.Equal(t, updateReq.IncludeFriends, updatedBan.IncludeFriends)
	require.Equal(t, updateReq.EvadeOk, updatedBan.EvadeOk)
	require.True(t, updatedBan.ValidUntil.After(fetchedBan.ValidUntil))

	// Get the ban by ban_id
	var banByBanID domain.BannedPerson
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/steam/%d?deleted=true", updatedBan.BanID),
		nil, http.StatusOK, &authTokens{user: modCreds}, &banByBanID)
	require.EqualExportedValues(t, updatedBan, banByBanID)

	// Get the same ban when querying a users active ban
	var banBySteamID domain.BannedPerson
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/steamid/%d", target.SteamID.Int64()),
		nil, http.StatusOK, &authTokens{user: modCreds}, &banBySteamID)
	require.EqualExportedValues(t, updatedBan, banBySteamID)

	// Delete the ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/steam/%d", banBySteamID.BanID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusOK, &authTokens{user: modCreds})

	// Ensure it was deleted
	testEndpoint(t, router, http.MethodGet, fmt.Sprintf("/api/bans/steam/%d", banBySteamID.BanID),
		nil, http.StatusNotFound, &authTokens{user: modCreds})

	// Try to delete non existent bam
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/steam/%d", banBySteamID.BanID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusNotFound, &authTokens{user: modCreds})
}

func TestBansSteamPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/stats",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/steam",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		// {
		//	path:   "/api/bans/steamid/1",
		//	method: http.MethodGet,
		//	code:   http.StatusForbidden,
		//	levels: authed,
		// },
		{
			path:   "/api/bans/steam/create",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/steam/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/steam/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/steam/1/status",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		// {
		//	path:   "/api/bans/steam/1",
		//	method: http.MethodGet,
		//	code:   http.StatusForbidden,
		//	levels: moderators,
		// },
		{
			path:   "/api/sourcebans/1",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
	})
}
