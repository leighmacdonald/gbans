package ban_test

import (
	"fmt"
	"net/http"
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
	tokens := tests.AuthTokens{}
	var r ban.Ban
	for _, sid := range []steamid.SteamID{tests.GuestSID, tests.UserSID} {
		tests.EndpointReceiver(t, router, "POST", "/api/bans", ban.Opts{
			SourceID: tests.OwnerSID, TargetID: sid, Duration: duration.FromTimeDuration(time.Hour * 10),
			BanType: ban.Banned, Reason: ban.Cheating, Origin: ban.System,
		}, http.StatusCreated, &tokens, &r)
		time.Sleep(time.Second)
	}
	require.Positive(t, r.BanID)
	require.Equal(t, tests.OwnerSID, r.SourceID)
	require.Equal(t, tests.UserSID, r.TargetID)
	tests.Endpoint(t, router, "POST", "/api/bans", ban.Opts{
		SourceID: tests.OwnerSID, TargetID: r.TargetID, Duration: duration.FromTimeDuration(time.Hour * 10),
		BanType: ban.Banned, Reason: ban.Cheating, Origin: ban.System,
	}, http.StatusConflict, &tokens)

	var loadedBans []ban.Ban
	tests.EndpointReceiver(t, router, "GET", "/api/bans", ban.RequestQueryOpts{}, http.StatusOK, &tokens, &loadedBans)
	require.Len(t, loadedBans, 2)

	tests.Endpoint(t, router, "DELETE", fmt.Sprintf("/api/ban/%d", r.BanID), ban.RequestUnban{UnbanReasonText: "test reason"}, http.StatusOK, &tokens)
}
