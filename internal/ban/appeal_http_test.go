package ban_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

func TestHTTPAppeal(t *testing.T) {
	var (
		router  = fixture.CreateRouter()
		bans    = ban.NewBans(ban.NewRepository(fixture.Database, fixture.Persons), fixture.Persons, fixture.Config, nil, notification.NewNullNotifications())
		persons = person.NewPersons(
			person.NewRepository(fixture.Config.Config(), fixture.Database),
			steamid.New(tests.OwnerSID),
			fixture.TFApi)
		appeals = ban.NewAppeals(ban.NewAppealRepository(fixture.Database), bans, persons, fixture.Config, notification.NewNullNotifications())
		target  = steamid.RandSID64()
	)

	ban.NewAppealHandler(router, appeals, &tests.StaticAuthenticator{
		Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin),
	})

	testBan, errTestBan := bans.Create(t.Context(), ban.Opts{
		SourceID: tests.OwnerSID, TargetID: target, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: ban.Banned, Reason: ban.Cheating, Origin: ban.System,
	})

	require.NoError(t, errTestBan)

	// Check for no messages
	var banMessages []ban.AppealMessage
	tests.GetOK(t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), &banMessages)
	require.Empty(t, banMessages)

	// Create a message
	newMessage := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	var createdMessage ban.AppealMessage
	tests.PostCreated(t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage, &createdMessage)
	require.Equal(t, newMessage.BodyMD, createdMessage.MessageMD)

	// Try and create a message as non target user or mod
	// TODO fix other users being allowed
	// tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage, http.StatusForbidden, tokens, &createdMessage)

	// Get appeals
	var appealOverviews []ban.AppealOverview
	tests.PostOK(t, router, "/api/appeals", ban.AppealQueryFilter{Deleted: false}, &appealOverviews)
	require.NotEmpty(t, appealOverviews)

	// Get messages
	tests.GetOK(t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), &banMessages)
	require.NotEmpty(t, banMessages)

	// Edit the message
	editMessage := ban.RequestMessageBodyMD{BodyMD: createdMessage.MessageMD + "x"}
	var editedMessage ban.AppealMessage
	tests.PostOK(t, router, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), editMessage, &editedMessage)
	require.Equal(t, editMessage.BodyMD, editedMessage.MessageMD)

	// Delete the message
	tests.DeleteOK(t, router, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), nil)

	// Confirm delete
	tests.GetOK(t, router, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), &banMessages)
	require.Empty(t, banMessages)
}
