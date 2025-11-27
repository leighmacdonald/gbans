package news_test

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestNewsHTTP(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)}
		router        = fixture.CreateRouter()
	)

	news.NewNewsHandler(router, news.NewNews(news.NewRepository(fixture.Database),
		notification.NewDiscard(), ""), authenticator)

	// No news yet
	require.Empty(t, tests.GetGOK[[]news.Article](t, router, "/api/news_latest"))

	// Add one
	req := news.EditRequest{
		Title:       stringutil.SecureRandomString(10),
		BodyMD:      stringutil.SecureRandomString(100),
		IsPublished: true,
	}
	saved := tests.PostGCreated[news.Article](t, router, "/api/news", req)
	require.Positive(t, saved.NewsID)
	require.Equal(t, req.Title, saved.Title)
	require.Equal(t, req.BodyMD, saved.BodyMD)
	require.Equal(t, req.IsPublished, saved.IsPublished)

	// Update it
	updateReq := req
	updateReq.BodyMD = stringutil.SecureRandomString(100)
	updated := tests.PostGOK[news.Article](t, router, fmt.Sprintf("/api/news/%d", saved.NewsID), updateReq)
	require.Equal(t, updateReq.Title, updated.Title)
	require.Equal(t, updateReq.BodyMD, updated.BodyMD)
	require.Equal(t, updateReq.IsPublished, updated.IsPublished)

	// Fetch it
	require.Len(t, tests.GetGOK[[]news.Article](t, router, "/api/news_latest"), 1)
	require.Len(t, tests.GetGOK[[]news.Article](t, router, "/api/news_all"), 1)

	// Delete it
	tests.DeleteOK(t, router, fmt.Sprintf("/api/news/%d", saved.NewsID), nil)

	// Confirm delete
	require.Empty(t, tests.GetGOK[[]news.Article](t, router, "/api/news_latest"))
	require.Empty(t, tests.GetGOK[[]news.Article](t, router, "/api/news_all"))
}
