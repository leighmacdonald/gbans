package ban_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

func TestHTTPBan(t *testing.T) {
	router := fixture.CreateRouter()
	bans := ban.NewBans(ban.NewRepository(fixture.Database, fixture.Persons), fixture.Persons, fixture.Config, nil, notification.NewNullNotifications())
	ban.NewHandlerBans(router, bans, fixture.Config, &tests.StaticAuthenticator{
		Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin),
	})
	var createdBan ban.Ban
	for _, sid := range []steamid.SteamID{tests.GuestSID, tests.UserSID} {
		tests.PostCreated(t, router, "/api/bans", ban.Opts{
			SourceID: tests.OwnerSID, TargetID: sid, Duration: duration.FromTimeDuration(time.Hour * 10),
			BanType: ban.Banned, Reason: ban.Cheating, Origin: ban.System,
		}, &createdBan)
	}
	require.Positive(t, createdBan.BanID)
	require.Equal(t, tests.OwnerSID, createdBan.SourceID)
	require.Equal(t, tests.UserSID, createdBan.TargetID)
	tests.PostConflict(t, router, "/api/bans", ban.Opts{
		SourceID: tests.OwnerSID, TargetID: createdBan.TargetID, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: ban.Banned, Reason: ban.Cheating, Origin: ban.System,
	})

	var loadedBans []ban.Ban
	tests.GetOK(t, router, "/api/bans", ban.RequestQueryOpts{AppealState: ptr(int(ban.AnyState))}, &loadedBans)
	require.Len(t, loadedBans, 2)

	tests.DeleteOK(t, router, fmt.Sprintf("/api/ban/%d", createdBan.BanID), ban.RequestUnban{UnbanReasonText: "test reason"})

	tests.GetOK(t, router, "/api/bans", ban.RequestQueryOpts{AppealState: ptr(int(ban.AnyState))}, &loadedBans)
	require.Len(t, loadedBans, 1)

	tests.GetOK(t, router, "/api/bans", ban.RequestQueryOpts{AppealState: ptr(int(ban.AnyState)), Deleted: true}, &loadedBans)
	require.Len(t, loadedBans, 2)

	var stats ban.Stats
	tests.GetOK(t, router, "/api/stats", &stats)
	require.Equal(t, 2, stats.BansTotal)

	var single ban.Ban
	tests.GetOK(t, router, fmt.Sprintf("/api/ban/%d", loadedBans[0].BanID), &single)
	require.Equal(t, loadedBans[0], single)

	// var update2 ban.Ban
	// tests.EndpointReceiver(t, router, "POST", fmt.Sprintf("/api/ban/%d", loadedBans[0].BanID), ban.RequestBanUpdate{
	// 	Reason: int(ban.BotHost),
	// }, http.StatusOK, &update2)
	// require.Equal(t, ban.BotHost, update2.Reason)

	tests.PostAccepted(t, router, fmt.Sprintf("/api/ban/%d/status", loadedBans[0].BanID), ban.SetStatusReq{AppealState: ban.Accepted})
}

func ptr[T any](v T) *T {
	return &v
}
