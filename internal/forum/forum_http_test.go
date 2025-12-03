package forum_test

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestCategories(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{}
		router        = fixture.CreateRouter()
		forumsUC      = forum.New(forum.NewRepository(fixture.Database), fixture.Config, notification.NewDiscard())
	)

	forum.NewForumHandler(router, authenticator, forumsUC)

	// Get the existing categories (empty)
	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)
	overview := tests.GetGOK[forum.Overview](t, router, "/api/forum/overview")
	require.Empty(t, overview.Categories)

	// Create a category
	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	req := forum.CategoryRequest{
		Title:       stringutil.SecureRandomString(10),
		Description: stringutil.SecureRandomString(100),
		Ordering:    1,
	}
	category := tests.PostGCreated[forum.Category](t, router, "/api/forum/category", req)
	require.Positive(t, category.ForumCategoryID)
	require.Equal(t, req.Title, category.Title)
	require.Equal(t, req.Description, category.Description)
	require.Equal(t, req.Ordering, category.Ordering)

	// Make sure it saved
	overview1 := tests.GetGOK[forum.Overview](t, router, "/api/forum/overview")
	require.NotEmpty(t, overview1.Categories)

	// Get a category by id
	cat := tests.GetGOK[forum.Category](t, router, fmt.Sprintf("/api/forum/category/%d", overview1.Categories[0].ForumCategoryID))
	require.Equal(t, overview1.Categories[0].ForumCategoryID, cat.ForumCategoryID)

	// Updaate it
	cat.Title += stringutil.SecureRandomString(3)
	cat.Description += stringutil.SecureRandomString(3)
	catUpdate := tests.PostGOK[forum.Category](t, router, fmt.Sprintf("/api/forum/category/%d", cat.ForumCategoryID), cat)
	require.Equal(t, cat.Title, catUpdate.Title)
	require.Equal(t, cat.Description, catUpdate.Description)
}

func TestForums(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{
			Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator),
		}
		router   = fixture.CreateRouter()
		forumsUC = forum.New(forum.NewRepository(fixture.Database), fixture.Config, notification.NewDiscard())
	)

	forum.NewForumHandler(router, authenticator, forumsUC)
	cat := &forum.Category{
		Title:       stringutil.SecureRandomString(10),
		Description: stringutil.SecureRandomString(100),
		Ordering:    1,
	}
	require.NoError(t, forumsUC.CategorySave(t.Context(), cat))

	// Create forum
	req := forum.CreateForumRequest{
		Title:           stringutil.SecureRandomString(10),
		Description:     stringutil.SecureRandomString(100),
		Ordering:        1,
		ForumCategoryID: cat.ForumCategoryID,
		PermissionLevel: permission.Guest,
	}
	forum1 := tests.PostGCreated[forum.Forum](t, router, "/api/forum/forum", req)
	require.Positive(t, forum1.ForumID)

	// Update forum
	forum1.Title += stringutil.SecureRandomString(3)
	forum1.Description += stringutil.SecureRandomString(3)
	forum1.Ordering++
	forum2 := tests.PostGOK[forum.Forum](t, router, fmt.Sprintf("/api/forum/forum/%d", forum1.ForumID), forum1)
	require.Equal(t, forum1.Title, forum2.Title)
	require.Equal(t, forum1.Description, forum2.Description)
	require.Equal(t, forum1.Ordering, forum2.Ordering)
}

func TestThreads(t *testing.T) {
	// TODO tests permissions.
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		router        = fixture.CreateRouter()
		forumsUC      = forum.New(forum.NewRepository(fixture.Database), fixture.Config, notification.NewDiscard())
	)
	forum.NewForumHandler(router, authenticator, forumsUC)

	cat := &forum.Category{
		Title:       stringutil.SecureRandomString(10),
		Description: stringutil.SecureRandomString(100),
		Ordering:    1,
	}
	require.NoError(t, forumsUC.CategorySave(t.Context(), cat))
	testForum := cat.NewForum(stringutil.SecureRandomString(10), stringutil.SecureRandomString(100))
	require.NoError(t, forumsUC.ForumSave(t.Context(), &testForum))

	// Create a new thread
	req := forum.CreateThreadRequest{
		Title:  stringutil.SecureRandomString(10),
		BodyMD: stringutil.SecureRandomString(100),
		Sticky: true,
		Locked: true,
	}
	thread := tests.PostGCreated[forum.ThreadWithMessage](t, router, fmt.Sprintf("/api/forum/forum/%d/thread", testForum.ForumID), req)
	require.Positive(t, thread.ForumThreadID)

	threadUpdate := forum.ThreadUpdate{
		Title:  thread.Title + stringutil.SecureRandomString(3),
		Locked: !thread.Locked,
		Sticky: !thread.Sticky,
	}
	updatedThread := tests.PostGOK[forum.Thread](t, router, fmt.Sprintf("/api/forum/thread/%d", thread.ForumThreadID), threadUpdate)
	require.Equal(t, threadUpdate.Title, updatedThread.Title)
	require.Equal(t, threadUpdate.Locked, updatedThread.Locked)
	require.Equal(t, threadUpdate.Sticky, updatedThread.Sticky)

	// Update thread parent message
	update := forum.MessageUpdate{BodyMD: thread.Message.BodyMD + stringutil.SecureRandomString(3)}
	updated := tests.PostGOK[forum.Message](t, router, fmt.Sprintf("/api/forum/message/%d", thread.Message.ForumMessageID), update)
	require.Equal(t, update.BodyMD, updated.BodyMD)

	// Add a reply
	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	reply := forum.ThreadReply{BodyMD: stringutil.SecureRandomString(100)}
	replyMessage := tests.PostGCreated[forum.Message](t, router, fmt.Sprintf("/api/forum/thread/%d/message", thread.ForumThreadID), reply)
	require.Positive(t, replyMessage.ForumMessageID)
	require.Equal(t, reply.BodyMD, replyMessage.BodyMD)

	// Get messages
	require.Len(t, tests.PostGOK[[]forum.Message](t, router, "/api/forum/messages", forum.ThreadMessagesQuery{Deleted: false, ForumThreadID: thread.ForumThreadID}), 2)

	// Recent messages
	require.Len(t, tests.GetGOK[[]forum.Message](t, router, "/api/forum/messages/recent"), 1)

	// Delete the reply
	tests.DeleteOK(t, router, fmt.Sprintf("/api/forum/message/%d", replyMessage.ForumMessageID), nil)

	// Get messages again -1
	require.Len(t, tests.PostGOK[[]forum.Message](t, router, "/api/forum/messages", forum.ThreadMessagesQuery{Deleted: false, ForumThreadID: thread.ForumThreadID}), 1)

	// Delete thread
	tests.DeleteOK(t, router, fmt.Sprintf("/api/forum/thread/%d", thread.ForumThreadID), nil)

	// Try to get deleted thread
	tests.GetNotFound(t, router, fmt.Sprintf("/api/forum/thread/%d", thread.ForumThreadID))
}
