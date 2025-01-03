package tests_test

import (
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestGetWikiPageBySlugMissing(t *testing.T) {
	t.Parallel()

	router := testRouter()

	testEndpoint(t, router, http.MethodGet, "/api/wiki/slug/home", nil, http.StatusNotFound, nil)
}

func TestSaveWikiPageBySlugUnauthed(t *testing.T) {
	t.Parallel()

	router := testRouter()

	page := domain.NewWikiPage(stringutil.SecureRandomString(10), stringutil.SecureRandomString(500))

	testEndpoint(t, router, http.MethodPost, "/api/wiki/slug", page, http.StatusForbidden, nil)
}

func TestSaveWikiPageBySlugAuthed(t *testing.T) {
	t.Parallel()

	router := testRouter()
	tokens := loginUser(getModerator())

	page := domain.NewWikiPage(stringutil.SecureRandomString(10), stringutil.SecureRandomString(500))

	var createdPage domain.WikiPage
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/wiki/slug", page, http.StatusCreated, &authTokens{user: tokens}, &createdPage)

	var receivedPage domain.WikiPage
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/wiki/slug/"+page.Slug, page, http.StatusOK, &authTokens{user: tokens}, &receivedPage)
	require.Equal(t, page.BodyMD, receivedPage.BodyMD)
}
