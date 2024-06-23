package test_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestAppeal(t *testing.T) {
	router := testRouter()
	target := getUser()
	targetAuth := loginUser(target)

	mod := getModerator()
	modAuth := loginUser(mod)

	// Create a valid ban_id
	bannedPerson, errBan := banSteamUC.Ban(context.Background(), mod, domain.System, domain.RequestBanSteamCreate{
		SourceIDField:  domain.SourceIDField{SourceID: mod.SteamID.String()},
		TargetIDField:  domain.TargetIDField{TargetID: target.SteamID.String()},
		Duration:       "1d",
		BanType:        domain.Banned,
		Reason:         domain.Cheating,
		ReasonText:     "",
		Note:           "notes",
		ReportID:       0,
		DemoName:       "demo-test.dem",
		DemoTick:       100,
		IncludeFriends: true,
		EvadeOk:        true,
	})
	require.NoError(t, errBan)

	// Check for no messages
	var banMessages []domain.BanAppealMessage
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), nil, http.StatusOK, targetAuth, &banMessages)
	require.Empty(t, banMessages)

	// Create a message
	newMessage := domain.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	var createdMessage domain.BanAppealMessage
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), newMessage, http.StatusCreated, targetAuth, &createdMessage)
	require.Equal(t, newMessage.BodyMD, createdMessage.MessageMD)

	// Try and create a message as non target user or mod
	// TODO fix other users being allowed
	// testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), newMessage, http.StatusForbidden, loginUser(getUser()), &createdMessage)

	// Get appeals
	var appeals []domain.AppealOverview
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/appeals", domain.AppealQueryFilter{Deleted: false}, http.StatusOK, modAuth, &appeals)
	require.NotEmpty(t, appeals)

	// Get messages
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), nil, http.StatusOK, modAuth, &banMessages)
	require.NotEmpty(t, banMessages)

	// Edit the message
	editMessage := domain.RequestMessageBodyMD{BodyMD: createdMessage.MessageMD + "x"}
	var editedMessage domain.BanAppealMessage
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), editMessage, http.StatusOK, modAuth, &editedMessage)
	require.Equal(t, editMessage.BodyMD, editedMessage.MessageMD)

	// Delete the message
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), nil, http.StatusOK, modAuth)

	// Confirm delete
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), nil, http.StatusOK, modAuth, &banMessages)
	require.Empty(t, banMessages)
}

func TestAppealPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/bans/1/messages",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/bans/1/messages",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/bans/message/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/bans/message/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: authed,
		},
		{
			path:   "/api/appeals",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
