package ban_test

import (
	"testing"
)

func TestAppeal(t *testing.T) {
	// router := testRouter()

	// targetAuth := loginUser(testTarget)

	// mod := getModerator()
	// modAuth := loginUser(mod)

	// // Check for no messages
	// var banMessages []ban.AppealMessage
	// testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: targetAuth}, &banMessages)
	// require.Empty(t, banMessages)

	// // Create a message
	// newMessage := ban.RequestMessageBodyMD{BodyMD: stringutil.SecureRandomString(100)}
	// var createdMessage ban.AppealMessage
	// testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), newMessage, http.StatusCreated, &authTokens{user: targetAuth}, &createdMessage)
	// require.Equal(t, newMessage.BodyMD, createdMessage.MessageMD)

	// // Try and create a message as non target user or mod
	// // TODO fix other users being allowed
	// // testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/%d/messages", bannedPerson.BanID), newMessage, http.StatusForbidden, loginUser(getUser()), &createdMessage)

	// // Get appeals
	// var appeals []ban.AppealOverview
	// testEndpointWithReceiver(t, router, http.MethodPost, "/api/appeals", ban.AppealQueryFilter{Deleted: false}, http.StatusOK, &authTokens{user: modAuth}, &appeals)
	// require.NotEmpty(t, appeals)

	// // Get messages
	// testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: modAuth}, &banMessages)
	// require.NotEmpty(t, banMessages)

	// // Edit the message
	// editMessage := ban.RequestMessageBodyMD{BodyMD: createdMessage.MessageMD + "x"}
	// var editedMessage ban.AppealMessage
	// testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), editMessage, http.StatusOK, &authTokens{user: modAuth}, &editedMessage)
	// require.Equal(t, editMessage.BodyMD, editedMessage.MessageMD)

	// // Delete the message
	// testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/bans/message/%d", createdMessage.BanMessageID), nil, http.StatusOK, &authTokens{user: modAuth})

	// // Confirm delete
	// testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/bans/%d/messages", testBan.BanID), nil, http.StatusOK, &authTokens{user: modAuth}, &banMessages)
	// require.Empty(t, banMessages)
}
