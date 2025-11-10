package ban_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

func TestHTTPAppeal(t *testing.T) {
	var (
		router  = fixture.CreateRouter()
		persons = person.NewPersons(
			person.NewRepository(fixture.Database, true),
			steamid.New(tests.OwnerSID),
			fixture.TFApi)
		assets  = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos   = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database), assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		reports = ban.NewReports(ban.NewReportRepository(fixture.Database), persons, demos, fixture.TFApi, notification.NewNullNotifications(), "", "")
		bans    = ban.NewBans(ban.NewRepository(fixture.Database, fixture.Persons), fixture.Persons,
			fixture.Config.Config().Discord.BanLogChannelID,
			steamid.New(fixture.Config.Config().Owner), reports, notification.NewNullNotifications())
		appeals = ban.NewAppeals(ban.NewAppealRepository(fixture.Database), bans, persons, notification.NewNullNotifications())
		target  = steamid.RandSID64()
	)

	ban.NewAppealHandler(router, &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)}, appeals)

	testBan, errTestBan := bans.Create(t.Context(), ban.Opts{
		SourceID: tests.OwnerSID, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: bantype.Banned, Reason: reason.Cheating, Origin: ban.System,
	})

	require.NoError(t, errTestBan)

	// Check for no messages
	banMessages := tests.GetGOK[[]ban.AppealMessage](t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID))
	require.Empty(t, banMessages)

	// Create a message
	newMessage := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	createdMessage := tests.PostGCreated[ban.AppealMessage](t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage)
	require.Equal(t, newMessage.BodyMD, createdMessage.MessageMD)

	// Try and create a message as non target user or mod
	// TODO fix other users being allowed
	// tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage, http.StatusForbidden, tokens, &createdMessage)

	// Get appeals
	require.NotEmpty(t, tests.PostGOK[[]ban.AppealOverview](t, router, "/api/appeals", ban.AppealQueryFilter{Deleted: false}))

	// Get messages
	require.NotEmpty(t, tests.GetGOK[[]ban.AppealMessage](t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID)))

	// Edit the message
	editMessage := ban.RequestMessageBodyMD{BodyMD: createdMessage.MessageMD + "x"}
	editedMessage := tests.PostGOK[ban.AppealMessage](t, router, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), editMessage)
	require.Equal(t, editMessage.BodyMD, editedMessage.MessageMD)

	// Delete the message
	tests.DeleteOK(t, router, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), nil)

	// Confirm delete
	require.Empty(t, tests.GetGOK[[]ban.AppealMessage](t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID)))
}
