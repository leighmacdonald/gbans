package tests_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBansASN(t *testing.T) {
	router := testRouter()
	mod := getModerator()
	modCreds := loginUser(mod)
	target := getUser()

	// Ensure no bans exist
	var bansEmpty []domain.BanASN
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/asn", nil, http.StatusOK, &authTokens{user: modCreds}, &bansEmpty)
	require.Empty(t, bansEmpty)

	// Create a ban
	banReq := domain.RequestBanASNCreate{
		SourceIDField: domain.SourceIDField{SourceID: mod.SteamID.String()},
		TargetIDField: domain.TargetIDField{TargetID: target.SteamID.String()},
		Duration:      "1d",
		Reason:        domain.Cheating,
		ReasonText:    "",
		Note:          "notes",
		ASNum:         1234,
	}

	var fetchedBan domain.BannedASNPerson
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/bans/asn/create", banReq, http.StatusCreated, &authTokens{user: modCreds}, &fetchedBan)

	require.Equal(t, banReq.SourceID, fetchedBan.SourceID.String())
	require.Equal(t, banReq.TargetID, fetchedBan.TargetID.String())
	require.True(t, fetchedBan.ValidUntil.After(time.Now()))
	require.Equal(t, banReq.ASNum, fetchedBan.ASNum)
	require.Equal(t, banReq.Reason, fetchedBan.Reason)
	require.Equal(t, banReq.ReasonText, fetchedBan.ReasonText)
	require.Equal(t, banReq.Note, fetchedBan.Note)

	// Ensure it's in the ban collection
	var bans []domain.BanASN
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/bans/asn", nil, http.StatusOK, &authTokens{user: modCreds}, &bans)
	require.NotEmpty(t, bans)

	updateReq := domain.RequestBanASNUpdate{
		SourceIDField: domain.SourceIDField{SourceID: fetchedBan.SourceID.String()},
		TargetIDField: domain.TargetIDField{TargetID: fetchedBan.TargetID.String()},
		Reason:        domain.Custom,
		ReasonText:    "blah",
		Note:          "edited",
		ASNum:         2345,
		ValidUntil:    fetchedBan.ValidUntil.Add(time.Second * 10),
	}

	// Update the ban
	var updatedBan domain.BannedASNPerson
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/asn/%d", fetchedBan.BanASNId),
		updateReq, http.StatusOK, &authTokens{user: modCreds}, &updatedBan)

	require.Equal(t, updateReq.TargetID, updatedBan.TargetID.String())
	require.Equal(t, updateReq.Reason, updatedBan.Reason)
	require.Equal(t, updateReq.ReasonText, updatedBan.ReasonText)
	require.Equal(t, updateReq.Note, updatedBan.Note)
	require.Equal(t, updateReq.ASNum, updatedBan.ASNum)
	require.True(t, updatedBan.ValidUntil.After(fetchedBan.ValidUntil))

	// Delete the ban
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/asn/%d", updatedBan.BanASNId),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusOK, &authTokens{user: modCreds})

	// Try to delete non existent bam
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/asn/%d", updatedBan.BanASNId),
		domain.RequestUnban{UnbanReasonText: "test unban"}, http.StatusNotFound, &authTokens{user: modCreds})
}

func TestBansASNPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/bans/asn",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/asn/create",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/asn/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/bans/asn/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
