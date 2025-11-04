package wiki_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestGetSlug(t *testing.T) {
	var (
		authenticator = &tests.StaticAuthenticator{}
		wuc           = wiki.NewWiki(wiki.NewRepository(fixture.Database))
		router        = fixture.CreateRouter()
		slug          = stringutil.SecureRandomString(10)
		page          = wiki.NewPage(slug, stringutil.SecureRandomString(1000))
	)

	wiki.NewWikiHandler(router, wuc, authenticator)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	tests.GetNotFound(t, router, "/api/wiki/slug/"+slug)

	saved, err := wuc.Save(t.Context(), page)
	require.NoError(t, err)
	require.Equal(t, page.Revision+1, saved.Revision)

	var fetched wiki.Page
	tests.GetOK(t, router, "/api/wiki/slug/"+slug, &fetched)
	require.Equal(t, saved.Revision, fetched.Revision)

	saved.PermissionLevel = permission.Moderator
	_, _ = wuc.Save(t.Context(), saved)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	tests.GetOK(t, router, "/api/wiki/slug/"+slug)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	tests.GetForbidden(t, router, "/api/wiki/slug/"+slug)
}

func TestPutSlug(t *testing.T) {
	var (
		authenticator = &tests.StaticAuthenticator{}
		wuc           = wiki.NewWiki(wiki.NewRepository(fixture.Database))
		router        = fixture.CreateRouter()
		page          = wiki.NewPage(stringutil.SecureRandomString(10), stringutil.SecureRandomString(1000))
	)
	wiki.NewWikiHandler(router, wuc, authenticator)

	var resp wiki.Page

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	tests.PutForbidden(t, router, "/api/wiki/slug/"+page.Slug, page)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)

	tests.PutOK(t, router, "/api/wiki/slug/"+page.Slug, page, &resp)
	require.Equal(t, page.Revision+1, resp.Revision)
}
