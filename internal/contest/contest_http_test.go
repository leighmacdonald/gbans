package contest_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/contest"
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

func TestContestsHTTP(t *testing.T) {
	var (
		assets        = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)}
		contests      = contest.NewContests(contest.NewRepository(fixture.Database))
		router        = fixture.CreateRouter()
	)

	contest.NewContestHandler(router, authenticator, contests, assets)

	require.Empty(t, tests.GetGOK[[]contest.Contest](t, router, "/api/contests"))

	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)

	reqPublicContest, _ := contest.NewContest(stringutil.SecureRandomString(10), stringutil.SecureRandomString(100), time.Now(), time.Now().Add(time.Hour*24), true)
	reqPrivContest, _ := contest.NewContest(stringutil.SecureRandomString(10), stringutil.SecureRandomString(100), time.Now(), time.Now().Add(time.Hour*24), false)

	public := tests.PostGOK[contest.Contest](t, router, "/api/contests", reqPublicContest)
	require.False(t, public.ContestID.IsNil())

	priv := tests.PostGOK[contest.Contest](t, router, "/api/contests", reqPrivContest)
	require.False(t, priv.ContestID.IsNil())

	// fetched := tests.GetGOK[contest.Contest](t, router, fmt.Sprintf("/api/contests/%s", public.ContestID.String()))
	// require.Equal(t, public.ContestID, fetched.ContestID)

	// require.Len(t, tests.GetGOK[[]contest.Contest](t, router, "/api/contests"), 2)

	// authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)
	// require.Len(t, tests.GetGOK[[]contest.Contest](t, router, "/api/contests"), 1)
}
