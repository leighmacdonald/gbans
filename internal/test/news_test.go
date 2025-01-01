package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestNews(t *testing.T) {
	router := testRouter()

	entry := domain.NewsEntry{
		Title:       stringutil.SecureRandomString(10),
		BodyMD:      stringutil.SecureRandomString(200),
		IsPublished: true,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	testEndpoint(t, router, http.MethodPost, "/api/news", entry, http.StatusForbidden, &authTokens{user: loginUser(getUser())})

	var newEntry domain.NewsEntry
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/news", entry, http.StatusCreated, &authTokens{user: loginUser(getModerator())}, &newEntry)

	testEndpoint(t, router, http.MethodPost, "/api/news_all", entry, http.StatusForbidden, &authTokens{user: loginUser(getUser())})

	var entries []domain.NewsEntry
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/news_all", entry, http.StatusOK, &authTokens{user: loginUser(getModerator())}, &entries)
	require.Len(t, entries, 1)

	edited := newEntry
	edited.BodyMD = stringutil.SecureRandomString(200)

	var receivedEdited domain.NewsEntry
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/news/%d", edited.NewsID), edited, http.StatusAccepted, &authTokens{user: loginUser(getUser())}, &receivedEdited)

	require.Equal(t, edited.Title, receivedEdited.Title)
	require.Equal(t, edited.BodyMD, receivedEdited.BodyMD)
	require.True(t, edited.UpdatedOn.Before(receivedEdited.UpdatedOn))

	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/news/%d", receivedEdited.NewsID), entry, http.StatusForbidden, &authTokens{user: loginUser(getUser())})
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/news/%d", receivedEdited.NewsID), entry, http.StatusOK, &authTokens{user: loginUser(getOwner())})

	testEndpointWithReceiver(t, router, http.MethodPost, "/api/news_latest", entry, http.StatusOK, &authTokens{user: loginUser(getUser())}, &entries)

	require.Empty(t, entries)
}
