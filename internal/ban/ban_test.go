package ban_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

// fixture provides a shared set of common dependencies that can be used for integration testing.
var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestBan(t *testing.T) {
	t.Parallel()
	var (
		assets = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos  = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database),
			assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		reports = ban.NewReports(ban.NewReportRepository(fixture.Database),
			person.NewPersons(person.NewRepository(fixture.Database, true), steamid.New(tests.OwnerSID), fixture.TFApi),
			demos, fixture.TFApi, notification.NewDiscard(), "")
		serversCase, _ = servers.New(servers.NewRepository(fixture.Database), nil, "")
		bans           = ban.New(ban.NewRepository(fixture.Database), fixture.Persons,
			fixture.Config.Config().Discord.BanLogChannelID, fixture.Config.Config().Discord.KickLogChannelID,
			steamid.New(fixture.Config.Config().Owner), reports, notification.NewDiscard(), serversCase, tests.EmptyIPProvider{})
		source = steamid.RandSID64()
		target = steamid.RandSID64()
	)

	newBan, err := bans.Create(t.Context(), ban.Opts{
		SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: bantype.Banned, Reason: reason.Cheating, Origin: ban.System,
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
		assets = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos  = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database),
			assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		reports = ban.NewReports(ban.NewReportRepository(fixture.Database),
			person.NewPersons(person.NewRepository(fixture.Database, true), steamid.New(tests.OwnerSID), fixture.TFApi),
			demos, fixture.TFApi, notification.NewDiscard(), "")
		serversCase, _ = servers.New(servers.NewRepository(fixture.Database), nil, "")
		bans           = ban.New(ban.NewRepository(fixture.Database), fixture.Persons,
			fixture.Config.Config().Discord.BanLogChannelID, fixture.Config.Config().Discord.KickLogChannelID,
			steamid.New(fixture.Config.Config().Owner), reports, notification.NewDiscard(), serversCase, tests.EmptyIPProvider{})
		source = steamid.RandSID64()
		target = steamid.RandSID64()
		opts   = []ban.Opts{
			{
				SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
				BanType: bantype.Banned, Reason: reason.Cheating, Origin: ban.System,
			},
			{
				SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
				BanType: bantype.Banned, Reason: reason.Cheating, Origin: ban.System,
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
		assets = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos  = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database),
			assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		reports = ban.NewReports(ban.NewReportRepository(fixture.Database),
			person.NewPersons(person.NewRepository(fixture.Database, true), steamid.New(tests.OwnerSID), fixture.TFApi),
			demos, fixture.TFApi, notification.NewDiscard(), "")
		serversCase, _ = servers.New(servers.NewRepository(fixture.Database), nil, "")
		bans           = ban.New(ban.NewRepository(fixture.Database), fixture.Persons,
			fixture.Config.Config().Discord.BanLogChannelID, fixture.Config.Config().Discord.KickLogChannelID,
			steamid.New(fixture.Config.Config().Owner), reports, notification.NewDiscard(), serversCase, tests.EmptyIPProvider{})
		source = steamid.RandSID64()
		target = steamid.RandSID64()
		author = fixture.CreateTestPerson(t.Context(), source, permission.Admin)
	)
	testBan, err := bans.Create(t.Context(), ban.Opts{
		SourceID: source, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: bantype.Banned, Reason: reason.Cheating, Origin: ban.System,
	})
	require.NoError(t, err)

	didUnban, errUnban := bans.Unban(t.Context(), testBan.TargetID, stringutil.SecureRandomString(20), author)
	require.NoError(t, errUnban)
	require.True(t, didUnban)
}
