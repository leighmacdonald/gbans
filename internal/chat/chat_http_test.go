package chat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestMessages(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		router        = fixture.CreateRouter()
		server        = fixture.CreateTestServer(t.Context())
		filters       = chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database), notification.NewDiscard(), fixture.Config.Config().Filters)
		chats         = chat.NewChat(chat.NewRepository(fixture.Database), fixture.Config.Config().Filters, filters,
			fixture.Persons, notification.NewDiscard(), func(_ context.Context, _ bool, _ chat.NewUserWarning) error { return nil })
	)

	chat.NewChatHandler(router, chats, authenticator)

	userA := fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
	userB := fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)

	const count = 50
	for number := range count {
		body := stringutil.SecureRandomString(100)
		sid := userA.SteamID
		name := userA.GetName()
		if number%2 != 0 {
			sid = userB.SteamID
			name = userB.GetName()
		}
		msg := chat.Message{
			SteamID:     sid,
			PersonaName: strings.ToValidUTF8(name, "_"),
			ServerName:  server.ShortName,
			ServerID:    server.ServerID,
			Body:        body,
			Team:        number%10 == 0,
			CreatedOn:   time.Now(),
		}
		require.NoError(t, chats.AddChatHistory(t.Context(), &msg))
	}
	req := chat.HistoryQueryFilter{}
	messages := tests.PostGOK[[]chat.QueryChatHistoryResult](t, router, "/api/messages", req)
	require.Len(t, messages, count)

	require.Len(t, tests.PostGOK[[]chat.QueryChatHistoryResult](t, router, "/api/messages", chat.HistoryQueryFilter{
		SourceIDField: httphelper.SourceIDField{SourceID: userA.SteamID.String()},
	}), count/2)

	contextMessages := tests.GetGOK[[]chat.QueryChatHistoryResult](t, router, fmt.Sprintf("/api/message/%d/context/5", messages[25].PersonMessageID))
	require.Len(t, contextMessages, 11)
	require.Equal(t, messages[25], contextMessages[5])
}
