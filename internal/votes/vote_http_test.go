package votes_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestVotesHTTP(t *testing.T) {
	var (
		auth   = &tests.StaticAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)}
		br     = broadcaster.New[logparse.EventType, logparse.ServerEvent]()
		server = fixture.CreateTestServer(t.Context())
		source = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		target = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		router = fixture.CreateRouter()
		vote   = votes.NewVotes(votes.NewRepository(fixture.Database), br, notification.NewNullNotifications(), "", fixture.Persons)
	)

	votes.NewVotesHandler(router, vote, auth)

	// Perm check
	tests.PostForbidden(t, router, "/api/votes", votes.Query{})

	// Fetch as mod
	auth.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	require.Empty(t, tests.PostGOK[httphelper.LazyResult[votes.Result]](t, router, "/api/votes", votes.Query{}).Data)

	// Add some entries and query
	require.NoError(t, vote.Add(t.Context(), source.SteamID, target.SteamID, target.GetName(), false, server.ServerID, logparse.VoteCodeFailNoOutnumberYes))
	require.NoError(t, vote.Add(t.Context(), source.SteamID, target.SteamID, target.GetName(), true, server.ServerID, logparse.VoteCodeFailNoOutnumberYes))
	require.Len(t, tests.PostGOK[httphelper.LazyResult[votes.Result]](t, router, "/api/votes", votes.Query{Success: -1}).Data, 2)
}
