package chat_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestFilters(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		router        = fixture.CreateRouter()
		wordfilters   = chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database), notification.NewDiscard(), fixture.Config.Config().Filters)
		chats         = chat.New(chat.NewRepository(fixture.Database), fixture.Config.Config().Filters, wordfilters,
			fixture.Persons, notification.NewDiscard(), func(_ context.Context, _ bool, _ chat.NewUserWarning) error { return nil })
	)

	chat.NewWordFilterHandler(router, authenticator, fixture.Config.Config().Filters, wordfilters, chats)

	// Make sure none exist yet
	require.Empty(t, tests.GetGOK[[]chat.Filter](t, router, "/api/filters"))

	// Add a filter
	req, _ := chat.NewFilter(tests.ModSID, "test", false, chat.FilterActionKick, "10s", 10)
	filter := tests.PostGOK[chat.Filter](t, router, "/api/filters", req)
	require.Positive(t, filter.FilterID)

	// Ensure it exists
	require.Len(t, tests.GetGOK[[]chat.Filter](t, router, "/api/filters"), 1)

	// Edit filter
	filter.Pattern = "asdf"
	updated := tests.PostGOK[chat.Filter](t, router, fmt.Sprintf("/api/filters/%d", filter.FilterID), filter)
	require.Equal(t, filter.Pattern, updated.Pattern)

	// Ensure it edited the existing entry
	require.Len(t, tests.GetGOK[[]chat.Filter](t, router, "/api/filters"), 1)

	// Delete it
	tests.DeleteOK(t, router, fmt.Sprintf("/api/filters/%d", filter.FilterID), nil)

	// Make sure it deleted
	require.Empty(t, tests.GetGOK[[]chat.Filter](t, router, "/api/filters"))
}
