package test_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBansSteamgroup(t *testing.T) {
	router := testRouter()
	mod := getModerator()
	modCreds := loginUser(mod)
	target := getUser()

	// Ensure no bans exist
	var bansEmpty []domain.BannedGroupPerson
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/group", nil, http.StatusOK, modCreds, &bansEmpty)
	require.Empty(t, bansEmpty)

	// Create a ban
	banReq := domain.RequestBanGroupCreate{
		TargetIDField:  domain.TargetIDField{TargetID: target.SteamID.String()},
		TargetGIDField: domain.TargetGIDField{GroupID: "103582791470708370"},
		Duration:       "1d",
		Note:           "notes",
	}

	var fetchedBan domain.BannedGroupPerson
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/bans/group/create", banReq, http.StatusCreated, modCreds, &fetchedBan)

	require.Equal(t, banReq.TargetIDField.TargetID, fetchedBan.TargetID.String())
	require.True(t, fetchedBan.ValidUntil.After(time.Now()))
	require.Equal(t, banReq.Note, fetchedBan.Note)
	require.Equal(t, banReq.GroupID, fetchedBan.GroupID.String())

	// Ensure it's in the ban collection
	var bans []domain.BanCIDR
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/group", nil, http.StatusOK, modCreds, &bans)
	require.NotEmpty(t, bans)

	updateReq := domain.RequestBanGroupUpdate{
		TargetIDField: domain.TargetIDField{TargetID: fetchedBan.TargetID.String()},
		Note:          "edited",
		ValidUntil:    fetchedBan.ValidUntil.Add(time.Second * 10),
	}

	// Update the ban
	var updatedBan domain.BannedGroupPerson
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/group/%d", fetchedBan.BanGroupID),
		updateReq, http.StatusOK, modCreds, &updatedBan)

	require.Equal(t, updateReq.TargetID, updatedBan.TargetID.String())
	require.Equal(t, updateReq.Note, updatedBan.Note)
	require.True(t, updatedBan.ValidUntil.After(fetchedBan.ValidUntil))

	// Delete the ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/group/%d", updatedBan.BanGroupID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusOK, modCreds)

	// Ensure it was deleted
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/group", nil, http.StatusOK, modCreds, &bans)
	require.NotContains(t, bans, updatedBan)

	// Try to delete non existent ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/group/%d", updatedBan.BanGroupID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusNotFound, modCreds)
}

func TestBansGroupPermissions(t *testing.T) {
	t.Parallel()

	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/bans/group",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/group/create",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/group/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/group/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
