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
		authenticator = &tests.StaticAuth{}
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

	fetched := tests.GetGOK[wiki.Page](t, router, "/api/wiki/slug/"+slug)
	require.Equal(t, saved.Revision, fetched.Revision)

	saved.PermissionLevel = permission.Moderator
	_, _ = wuc.Save(t.Context(), saved)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	tests.GetGOK[wiki.Page](t, router, "/api/wiki/slug/"+slug)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	tests.GetForbidden(t, router, "/api/wiki/slug/"+slug)
}

func TestPutSlug(t *testing.T) {
	var (
		authenticator = &tests.StaticAuth{}
		wuc           = wiki.NewWiki(wiki.NewRepository(fixture.Database))
		router        = fixture.CreateRouter()
		page          = wiki.NewPage(stringutil.SecureRandomString(10), stringutil.SecureRandomString(1000))
	)
	wiki.NewWikiHandler(router, wuc, authenticator)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	tests.PutForbidden(t, router, "/api/wiki/slug/"+page.Slug, page)

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	updated := tests.PutGOK[wiki.Page](t, router, "/api/wiki/slug/"+page.Slug, page)
	require.Equal(t, page.Revision+1, updated.Revision)
}
