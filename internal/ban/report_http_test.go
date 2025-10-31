package ban_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
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
			person.NewRepository(fixture.Config.Config(), fixture.Database),
			steamid.New(tests.OwnerSID),
			fixture.TFApi)
		// appeals = ban.NewAppeals(ban.NewAppealRepository(fixture.Database), bans, persons, fixture.Config, notification.NewNullNotifications())
		assets        = asset.NewAssets(asset.NewLocalRepository(fixture.Database, "./"))
		demo          = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database), assets, fixture.Config)
		reports       = ban.NewReports(ban.NewReportRepository(fixture.Database), fixture.Config, persons, demo, fixture.TFApi, notification.NewNullNotifications())
		tokens        = &tests.AuthTokens{}
		moderator     = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
		reporter      = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.User)
		target        = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		externalUser  = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		authenticator = &tests.StaticAuthenticator{Profile: moderator}
	)

	ban.NewReportHandler(router, reports, authenticator)

	// Create a report
	req := ban.RequestReportCreate{
		SourceID:        reporter.SteamID,
		TargetID:        target.SteamID,
		Description:     stringutil.SecureRandomString(100),
		Reason:          ban.Cheating,
		ReasonText:      "",
		DemoID:          0,
		DemoTick:        0,
		PersonMessageID: 0,
	}
	var report ban.Report
	tests.EndpointReceiver(t, router, http.MethodPost, "/api/report", req, http.StatusCreated, tokens, &report)
	require.Equal(t, req.SourceID, report.SourceID)
	require.Equal(t, req.TargetID, report.TargetID)
	require.Equal(t, req.Description, report.Description)
	require.Equal(t, req.Reason, report.Reason)
	require.Equal(t, req.ReasonText, report.ReasonText)
	require.Equal(t, req.DemoID, report.DemoID)
	require.Equal(t, req.DemoTick, report.DemoTick)
	require.Equal(t, req.PersonMessageID, report.PersonMessageID)

	// Make sure we can query it
	var fetched ban.Report
	tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d", report.ReportID), nil, http.StatusOK, tokens, &fetched)
	require.Equal(t, report, fetched)

	// Make sure we can query all
	var fetchedColl []ban.Report
	tests.EndpointReceiver(t, router, http.MethodGet, "/api/reports/user", nil, http.StatusOK, tokens, &fetchedColl)
	require.NotEmpty(t, fetchedColl)

	var fetchedModColl []ban.Report
	authenticator.Profile = moderator
	tests.EndpointReceiver(t, router, http.MethodPost, "/api/reports", ban.ReportQueryFilter{Deleted: true}, http.StatusOK, tokens, &fetchedModColl)
	require.NotEmpty(t, fetchedModColl)

	// Make sure others cant query other users reports
	authenticator.Profile = externalUser
	tests.EndpointReceiver(t, router, http.MethodGet, "/api/reports/user", nil, http.StatusOK, tokens, &fetchedColl)
	require.Empty(t, fetchedColl)

	// Change the status
	statusReq := ban.RequestReportStatusUpdate{Status: ban.ClosedWithAction}
	authenticator.Profile = moderator
	tests.Endpoint(t, router, http.MethodPost, fmt.Sprintf("/api/report_status/%d", report.ReportID), statusReq, http.StatusOK, tokens)

	tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d", report.ReportID), nil, http.StatusOK, tokens, &fetched)
	require.Equal(t, statusReq.Status, fetched.ReportStatus)

	// Get empty child messages
	var messages []ban.ReportMessage
	tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, tokens, &messages)
	require.Empty(t, messages)

	// Add a reply
	var fetchedMsg ban.ReportMessage
	msgReq := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/report/%d/messages", report.ReportID), msgReq, http.StatusCreated, tokens, &fetchedMsg)
	require.Equal(t, msgReq.BodyMD, fetchedMsg.MessageMD)

	// Get the reply
	tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, tokens, &messages)
	require.NotEmpty(t, messages)

	// Edit the reply
	editMsgReq := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	var edited ban.ReportMessage
	tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/report/message/%d", report.ReportID), editMsgReq, http.StatusOK, tokens, &edited)
	require.Equal(t, editMsgReq.BodyMD, edited.MessageMD)

	// Delete the message
	tests.Endpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/report/message/%d", fetchedMsg.ReportMessageID), nil, http.StatusOK, tokens)

	// Make sure it was deleted
	tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, tokens, &messages)
	require.Empty(t, messages)
}
