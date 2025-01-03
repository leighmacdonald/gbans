package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestWordFilter(t *testing.T) {
	router := testRouter()
	moderator := getModerator()
	creds := loginUser(moderator)

	// Shouldn't be filters already
	var filters []domain.Filter
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", nil, http.StatusOK, &authTokens{user: creds}, &filters)
	require.Empty(t, filters)

	// Create a filter
	req, errReq := domain.NewFilter(moderator.SteamID, "test", true, domain.Mute, "1d", 1)
	require.NoError(t, errReq)

	var created domain.Filter
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filters", req, http.StatusOK, &authTokens{user: creds}, &created)
	require.Positive(t, created.FilterID)

	// Check it was added
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", req, http.StatusOK, &authTokens{user: creds}, &filters)
	require.NotEmpty(t, filters)

	// Edit it
	edit := filters[0]
	edit.Pattern = "blah"
	edit.IsRegex = false

	var edited domain.Filter
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/filters/%d", edit.FilterID), edit, http.StatusOK, &authTokens{user: creds}, &edited)
	require.Equal(t, edit.FilterID, edited.FilterID)
	require.Equal(t, edit.AuthorID, edited.AuthorID)
	require.Equal(t, edit.Pattern, edited.Pattern)
	require.Equal(t, edit.IsRegex, edited.IsRegex)
	require.Equal(t, edit.IsEnabled, edited.IsEnabled)
	require.Equal(t, edit.Action, edited.Action)
	require.Equal(t, edit.Duration, edited.Duration)
	require.Equal(t, edit.TriggerCount, edited.TriggerCount)
	require.Equal(t, edit.Weight, edited.Weight)
	require.NotEqual(t, edit.UpdatedOn, edited.UpdatedOn)

	// Match it
	var matched []domain.Filter
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filter_match", domain.RequestQuery{Query: edited.Pattern}, http.StatusOK, &authTokens{user: creds}, &matched)
	require.NotEmpty(t, matched)
	require.Equal(t, matched[0].FilterID, edited.FilterID)

	// Delete it
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/filters/%d", edit.FilterID), req, http.StatusOK, &authTokens{user: creds})

	// Shouldn't match now
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filter_match", domain.RequestQuery{Query: edited.Pattern}, http.StatusOK, &authTokens{user: creds}, &matched)
	require.Empty(t, matched)

	// Make sure it was deleted
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", nil, http.StatusOK, &authTokens{user: creds}, &filters)
	require.Empty(t, filters)
}

func TestWordFilterPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/filters/query",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/filters/state",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/filters",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/filters/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/filters/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: moderators,
		},
		{
			path:   "/api/filter_match",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
