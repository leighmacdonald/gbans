package ban_test

import (
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/database"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	os.Exit(m.Run())
}

func TestBan(t *testing.T) {
	t.Parallel()
	var (
		br     = ban.NewRepository(fixture.Database, fixture.Persons)
		bans   = ban.NewBans(br, fixture.Persons, fixture.Config, nil, nil, notification.NewNullNotifications())
		source = steamid.RandSID64()
		target = steamid.RandSID64()
	)

	newBan, err := bans.Create(t.Context(), ban.Opts{
		SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: banDomain.Banned, Reason: banDomain.Cheating, Origin: banDomain.System,
	})
	require.NoError(t, err)
	require.Positive(t, newBan.BanID)

	fetched, errFetch := bans.QueryOne(t.Context(), ban.QueryOpts{TargetID: newBan.TargetID})
	require.NoError(t, errFetch)
	require.Equal(t, newBan, fetched)
}

func TestDuplicate(t *testing.T) {
	t.Parallel()
	var (
		br     = ban.NewRepository(fixture.Database, fixture.Persons)
		bans   = ban.NewBans(br, fixture.Persons, fixture.Config, nil, nil, notification.NewNullNotifications())
		source = steamid.RandSID64()
		target = steamid.RandSID64()
		opts   = []ban.Opts{
			{
				SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
				BanType: banDomain.Banned, Reason: banDomain.Cheating, Origin: banDomain.System,
			},
			{
				SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
				BanType: banDomain.Banned, Reason: banDomain.Cheating, Origin: banDomain.System,
			},
		}
	)

	for idx, opt := range opts {
		testBan, err := bans.Create(t.Context(), opt)
		if idx == 0 {
			require.NoError(t, err)
			require.Positive(t, testBan.BanID)
		} else {
			require.Error(t, database.ErrDuplicate)
		}
	}
}

func TestUnban(t *testing.T) {
	t.Parallel()
	var (
		br     = ban.NewRepository(fixture.Database, fixture.Persons)
		bans   = ban.NewBans(br, fixture.Persons, fixture.Config, nil, nil, notification.NewNullNotifications())
		source = steamid.RandSID64()
		target = steamid.RandSID64()
		author = fixture.CreateTestPerson(t.Context(), source)
	)
	testBan, err := bans.Create(t.Context(), ban.Opts{
		SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: banDomain.Banned, Reason: banDomain.Cheating, Origin: banDomain.System,
	})
	require.NoError(t, err)

	didUnban, errUnban := bans.Unban(t.Context(), testBan.TargetID, stringutil.SecureRandomString(20), author)
	require.NoError(t, errUnban)
	require.True(t, didUnban)
}
