package test_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBansCIDR(t *testing.T) {
	router := testRouter()
	mod := getModerator()
	modCreds := loginUser(mod)
	target := getUser()

	// Ensure no bans exist
	var bansEmpty []domain.BannedCIDRPerson
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/cidr", nil, http.StatusOK, modCreds, &bansEmpty)
	require.Empty(t, bansEmpty)

	// Create a ban
	banReq := domain.RequestBanCIDRCreate{
		TargetIDField: domain.TargetIDField{TargetID: target.SteamID.String()},
		Duration:      "1d",
		Note:          "notes",
		Reason:        domain.Cheating,
		ReasonText:    "",
		CIDR:          "5.5.5.0/24",
	}

	var fetchedBan domain.BannedCIDRPerson
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/bans/cidr/create", banReq, http.StatusCreated, modCreds, &fetchedBan)

	require.Equal(t, banReq.TargetIDField.TargetID, fetchedBan.TargetID.String())
	require.True(t, fetchedBan.ValidUntil.After(time.Now()))
	require.Equal(t, banReq.Reason, fetchedBan.Reason)
	require.Equal(t, banReq.ReasonText, fetchedBan.ReasonText)
	require.Equal(t, banReq.Note, fetchedBan.Note)
	require.Equal(t, banReq.CIDR, fetchedBan.CIDR)

	// Ensure it's in the ban collection
	var bans []domain.BanCIDR
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/cidr", nil, http.StatusOK, modCreds, &bans)
	require.NotEmpty(t, bans)

	updateReq := domain.RequestBanCIDRUpdate{
		TargetID:   fetchedBan.TargetID,
		Reason:     domain.Custom,
		ReasonText: "blah",
		Note:       "edited",
		CIDR:       "6.6.6.0/24",
		ValidUntil: fetchedBan.ValidUntil.Add(time.Second * 10),
	}

	// Update the ban
	var updatedBan domain.BannedCIDRPerson
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/cidr/%d", fetchedBan.NetID),
		updateReq, http.StatusOK, modCreds, &updatedBan)

	require.Equal(t, updateReq.TargetID, updatedBan.TargetID)
	require.Equal(t, updateReq.Reason, updatedBan.Reason)
	require.Equal(t, updateReq.ReasonText, updatedBan.ReasonText)
	require.Equal(t, updateReq.Note, updatedBan.Note)
	require.Equal(t, updateReq.CIDR, updatedBan.CIDR)
	require.True(t, updatedBan.ValidUntil.After(fetchedBan.ValidUntil))

	// Delete the ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/cidr/%d", updatedBan.NetID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusOK, modCreds)

	// Ensure it was deleted
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/cidr", nil, http.StatusOK, modCreds, &bans)
	require.NotContains(t, bans, updatedBan)

	// Try to delete non existent ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/cidr/%d", updatedBan.NetID),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusNotFound, modCreds)
}

func TestBansCIDRPermissions(t *testing.T) {
	t.Parallel()

	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/export/bans/valve/network",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/cidr",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/cidr/create",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/cidr/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/cidr/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
