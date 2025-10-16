package votes_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func TestVotes(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()
	events := broadcaster.New[logparse.EventType, logparse.ServerEvent]()

	server := testFixture.CreateTestServer(t.Context())
	p1 := testFixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
	p2 := testFixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.Guest)
	vuc := votes.NewVotes(votes.NewRepository(testFixture.Database), events, notification.NullNotifier{}, nil, nil)
	require.NoError(t, vuc.Add(t.Context(), p1.SteamID, p2.SteamID, "kick", false, server.ServerID, logparse.VoteCodeFailNoOutnumberYes))
}
