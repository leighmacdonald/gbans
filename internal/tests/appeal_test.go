package tests_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestAppeal(t *testing.T) {
	router := testRouter()

	targetAuth := loginUser(testTarget)

	mod := getModerator()
	modAuth := loginUser(mod)

	// Check for no messages
	var banMessages []ban.AppealMessage
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: targetAuth}, &banMessages)
	require.Empty(t, banMessages)

	// Create a message
	newMessage := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	var createdMessage ban.AppealMessage
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage, http.StatusCreated, &authTokens{user: targetAuth}, &createdMessage)
	require.Equal(t, newMessage.BodyMD, createdMessage.MessageMD)

	// Try and create a message as non target user or mod
	// TODO fix other users being allowed
	// testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), newMessage, http.StatusForbidden, loginUser(getUser()), &createdMessage)

	// Get appeals
	var appeals []ban.AppealOverview
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/appeals", ban.AppealQueryFilter{Deleted: false}, http.StatusOK, &authTokens{user: modAuth}, &appeals)
	require.NotEmpty(t, appeals)

	// Get messages
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: modAuth}, &banMessages)
	require.NotEmpty(t, banMessages)

	// Edit the message
	editMessage := ban.RequestMessageBodyMD{BodyMD: createdMessage.MessageMD + "x"}
	var editedMessage ban.AppealMessage
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), editMessage, http.StatusOK, &authTokens{user: modAuth}, &editedMessage)
	require.Equal(t, editMessage.BodyMD, editedMessage.MessageMD)

	// Delete the message
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), nil, http.StatusOK, &authTokens{user: modAuth})

	// Confirm delete
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: modAuth}, &banMessages)
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
