package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	router := testRouter()
	source := getUser()
	sourceCreds := loginUser(source)
	mods := loginUser(getModerator())
	otherUser := loginUser(getUser())
	target := getUser()

	// Create a report
	req := domain.RequestReportCreate{
		SourceID:        source.SteamID,
		TargetID:        target.SteamID,
		Description:     stringutil.SecureRandomString(100),
		Reason:          domain.Cheating,
		ReasonText:      "",
		DemoID:          0,
		DemoTick:        0,
		PersonMessageID: 0,
	}
	var report domain.Report
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/report", req, http.StatusCreated, &authTokens{user: sourceCreds}, &report)
	require.EqualValues(t, req.SourceID, report.SourceID)
	require.EqualValues(t, req.TargetID, report.TargetID)
	require.EqualValues(t, req.Description, report.Description)
	require.EqualValues(t, req.Reason, report.Reason)
	require.EqualValues(t, req.ReasonText, report.ReasonText)
	require.EqualValues(t, req.DemoID, report.DemoID)
	require.EqualValues(t, req.DemoTick, report.DemoTick)
	require.EqualValues(t, req.PersonMessageID, report.PersonMessageID)

	// Make sure we can query it
	var fetched domain.Report
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d", report.ReportID), nil, http.StatusOK, &authTokens{user: sourceCreds}, &fetched)
	require.EqualValues(t, report, fetched)

	// Make sure we can query all
	var fetchedColl []domain.Report
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/reports/user", nil, http.StatusOK, &authTokens{user: sourceCreds}, &fetchedColl)
	require.NotEmpty(t, fetchedColl)

	var fetchedModColl []domain.Report
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/reports", domain.ReportQueryFilter{Deleted: true}, http.StatusOK, &authTokens{user: mods}, &fetchedModColl)
	require.NotEmpty(t, fetchedModColl)

	// Make sure others cant query other users reports
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/reports/user", nil, http.StatusOK, &authTokens{user: otherUser}, &fetchedColl)
	require.Empty(t, fetchedColl)

	// Change the status
	statusReq := domain.RequestReportStatusUpdate{Status: domain.ClosedWithAction}
	testEndpoint(t, router, http.MethodPost, fmt.Sprintf("/api/report_status/%d", report.ReportID), statusReq, http.StatusOK, &authTokens{user: sourceCreds})

	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d", report.ReportID), nil, http.StatusOK, &authTokens{user: sourceCreds}, &fetched)
	require.Equal(t, statusReq.Status, fetched.ReportStatus)

	// Get empty child messages
	var messages []domain.ReportMessage
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, &authTokens{user: sourceCreds}, &messages)
	require.Empty(t, messages)

	// Add a reply
	var fetchedMsg domain.ReportMessage
	msgReq := domain.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/report/%d/messages", report.ReportID), msgReq, http.StatusCreated, &authTokens{user: sourceCreds}, &fetchedMsg)
	require.Equal(t, msgReq.BodyMD, fetchedMsg.MessageMD)

	// Get the reply
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, &authTokens{user: sourceCreds}, &messages)
	require.NotEmpty(t, messages)

	// Edit the reply
	editMsgReq := domain.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	var edited domain.ReportMessage
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/report/message/%d", report.ReportID), editMsgReq, http.StatusOK, &authTokens{user: sourceCreds}, &edited)
	require.Equal(t, editMsgReq.BodyMD, edited.MessageMD)

	// Delete the message
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/report/message/%d", fetchedMsg.ReportMessageID), nil, http.StatusOK, &authTokens{user: sourceCreds})

	// Make sure it was deleted
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/report/%d/messages", report.ReportID), nil, http.StatusOK, &authTokens{user: sourceCreds}, &messages)
	require.Empty(t, messages)
}

func TestReportPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/report",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report/1",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report_status/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report/1/messages",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report/1/messages",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report/message/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/report/message/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/reports/user",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/reports",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
