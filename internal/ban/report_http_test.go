package ban_test

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	var (
		router = fixture.CreateRouter()
		// br     = ban.NewRepository(fixture.Database, fixture.Persons)
		// bans    = ban.NewBans(br, fixture.Persons, fixture.Config, nil, notification.NewNullNotifications())
		persons = person.NewPersons(
			person.NewRepository(fixture.Database, true),
			steamid.New(tests.OwnerSID),
			fixture.TFApi)
		// appeals = ban.NewAppeals(ban.NewAppealRepository(fixture.Database), bans, persons, fixture.Config, notification.NewNullNotifications())
		assets        = asset.NewAssets(asset.NewLocalRepository(fixture.Database, "./"))
		demo          = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database), assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		reports       = ban.NewReports(ban.NewReportRepository(fixture.Database), persons, demo, fixture.TFApi, notification.NewNullNotifications(), "", "")
		moderator     = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
		reporter      = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.User)
		target        = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		externalUser  = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		authenticator = &tests.UserAuth{Profile: moderator}
	)

	ban.NewReportHandler(router, authenticator, reports)

	// Create a report
	req := ban.RequestReportCreate{
		SourceID:        reporter.SteamID,
		TargetID:        target.SteamID,
		Description:     stringutil.SecureRandomString(100),
		Reason:          reason.Cheating,
		ReasonText:      "",
		DemoID:          0,
		DemoTick:        0,
		PersonMessageID: 0,
	}

	report := tests.PostGCreated[ban.Report](t, router, "/api/report", req)
	require.Equal(t, req.SourceID, report.SourceID)
	require.Equal(t, req.TargetID, report.TargetID)
	require.Equal(t, req.Description, report.Description)
	require.Equal(t, req.Reason, report.Reason)
	require.Equal(t, req.ReasonText, report.ReasonText)
	require.Equal(t, req.DemoID, report.DemoID)
	require.Equal(t, req.DemoTick, report.DemoTick)
	require.Equal(t, req.PersonMessageID, report.PersonMessageID)

	// Make sure we can query it
	require.Equal(t, report, tests.GetGOK[ban.Report](t, router, fmt.Sprintf("/api/report/%d", report.ReportID)))

	// Make sure we can query all
	fetchedColl := tests.GetGOK[[]ban.Report](t, router, "/api/reports/user")
	require.NotEmpty(t, fetchedColl)

	authenticator.Profile = moderator
	require.NotEmpty(t, tests.PostGOK[[]ban.Report](t, router, "/api/reports", ban.ReportQueryFilter{Deleted: true}))

	// Make sure others cant query other users reports
	authenticator.Profile = externalUser
	require.Empty(t, tests.GetGOK[[]ban.Report](t, router, "/api/reports/user"))

	// Change the status
	statusReq := ban.RequestReportStatusUpdate{Status: ban.ClosedWithAction}
	authenticator.Profile = moderator
	tests.PostOK(t, router, fmt.Sprintf("/api/report_status/%d", report.ReportID), statusReq)

	fetched := tests.GetGOK[ban.Report](t, router, fmt.Sprintf("/api/report/%d", report.ReportID))
	require.Equal(t, statusReq.Status, fetched.ReportStatus)

	// Get empty child messages
	require.Empty(t, tests.GetGOK[[]ban.ReportMessage](t, router, fmt.Sprintf("/api/report/%d/messages", report.ReportID)))

	// Add a reply
	msgReq := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	fetchedMsg := tests.PostGCreated[ban.ReportMessage](t, router, fmt.Sprintf("/api/report/%d/messages", report.ReportID), msgReq)
	require.Equal(t, msgReq.BodyMD, fetchedMsg.MessageMD)

	// Get the reply

	require.NotEmpty(t, tests.GetGOK[[]ban.ReportMessage](t, router, fmt.Sprintf("/api/report/%d/messages", report.ReportID)))

	// Edit the reply
	editMsgReq := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	edited := tests.PostGOK[ban.ReportMessage](t, router, fmt.Sprintf("/api/report/message/%d", report.ReportID), editMsgReq)
	require.Equal(t, editMsgReq.BodyMD, edited.MessageMD)

	// Delete the message
	tests.DeleteOK(t, router, fmt.Sprintf("/api/report/message/%d", fetchedMsg.ReportMessageID), nil)

	// Make sure it was deleted
	require.Empty(t, tests.GetGOK[[]ban.ReportMessage](t, router, fmt.Sprintf("/api/report/%d/messages", report.ReportID)))
}
