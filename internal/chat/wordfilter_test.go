package chat_test

// func TestWordFilter(t *testing.T) {
// 	router := testRouter()
// 	moderator := getModerator()
// 	creds := loginUser(moderator)

// 	// Shouldn't be filters already
// 	var filters []chat.Filter
// 	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", nil, http.StatusOK, &authTokens{user: creds}, &filters)
// 	require.Empty(t, filters)

// 	// Create a filter
// 	req, errReq := chat.NewFilter(moderator.SteamID, "test", true, chat.FilterActionMute, "1d", 1)
// 	require.NoError(t, errReq)

// 	var created chat.Filter
// 	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filters", req, http.StatusOK, &authTokens{user: creds}, &created)
// 	require.Positive(t, created.FilterID)

// 	// Check it was added
// 	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", req, http.StatusOK, &authTokens{user: creds}, &filters)
// 	require.NotEmpty(t, filters)

// 	// Edit it
// 	edit := filters[0]
// 	edit.Pattern = "blah"
// 	edit.IsRegex = false

// 	var edited chat.Filter
// 	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/filters/%d", edit.FilterID), edit, http.StatusOK, &authTokens{user: creds}, &edited)
// 	require.Equal(t, edit.FilterID, edited.FilterID)
// 	require.Equal(t, edit.AuthorID, edited.AuthorID)
// 	require.Equal(t, edit.Pattern, edited.Pattern)
// 	require.Equal(t, edit.IsRegex, edited.IsRegex)
// 	require.Equal(t, edit.IsEnabled, edited.IsEnabled)
// 	require.Equal(t, edit.Action, edited.Action)
// 	require.Equal(t, edit.Duration, edited.Duration)
// 	require.Equal(t, edit.TriggerCount, edited.TriggerCount)
// 	require.Equal(t, edit.Weight, edited.Weight)
// 	require.NotEqual(t, edit.UpdatedOn, edited.UpdatedOn)

// 	// Match it
// 	var matched []chat.Filter
// 	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filter_match", httphelper.RequestQuery{Query: edited.Pattern}, http.StatusOK, &authTokens{user: creds}, &matched)
// 	require.NotEmpty(t, matched)
// 	require.Equal(t, matched[0].FilterID, edited.FilterID)

// 	// Delete it
// 	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/filters/%d", edit.FilterID), req, http.StatusOK, &authTokens{user: creds})

// 	// Shouldn't match now
// 	testEndpointWithReceiver(t, router, http.MethodPost, "/api/filter_match", httphelper.RequestQuery{Query: edited.Pattern}, http.StatusOK, &authTokens{user: creds}, &matched)
// 	require.Empty(t, matched)

// 	// Make sure it was deleted
// 	testEndpointWithReceiver(t, router, http.MethodGet, "/api/filters", nil, http.StatusOK, &authTokens{user: creds}, &filters)
// 	require.Empty(t, filters)
// }
