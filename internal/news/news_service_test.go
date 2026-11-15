package news_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/news"
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

func TestSaveStampsServerTimestamps(t *testing.T) {
	t.Parallel()

	svc := news.New(news.NewRepository(fixture.Database), discord.NewDiscard(&discord.Config{}), "")
	ctx := t.Context()

	before := time.Now()
	entry := &news.Article{Title: "Timestamp test", BodyMD: "Timestamp test body, long enough", IsPublished: false}
	require.NoError(t, svc.Save(ctx, entry))
	after := time.Now()

	require.Positive(t, entry.NewsID)
	require.True(t, !entry.CreatedOn.Before(before) && !entry.CreatedOn.After(after),
		"created_on %v outside save window [%v, %v]", entry.CreatedOn, before, after)
	require.True(t, !entry.UpdatedOn.Before(before) && !entry.UpdatedOn.After(after),
		"updated_on %v outside save window [%v, %v]", entry.UpdatedOn, before, after)

	// A re-save must preserve created_on and stamp a fresh updated_on.
	time.Sleep(20 * time.Millisecond)

	entry.Title = "Timestamp test (edited)"
	require.NoError(t, svc.Save(ctx, entry))

	fetched := news.Article{}
	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &fetched))
	require.True(t, fetched.CreatedOn.Equal(entry.CreatedOn.Truncate(time.Microsecond)),
		"created_on changed on edit: %v != %v", fetched.CreatedOn, entry.CreatedOn)
	require.True(t, fetched.UpdatedOn.After(fetched.CreatedOn),
		"updated_on %v not after created_on %v", fetched.UpdatedOn, fetched.CreatedOn)

	require.NoError(t, svc.DropNewsArticle(ctx, entry.NewsID))
}

type postedMessage struct {
	channelID string
	message   *discordgo.MessageSend
}

type targetedMessage struct {
	channelID string
	messageID string
	message   *discordgo.MessageSend
}

// recordingDiscord implements discord.Connection, recording channel
// interactions for assertions. With returnIDs false, SendWithID models a bot
// that is not running (it never returns a message id).
type recordingDiscord struct {
	sync.Mutex

	returnIDs bool
	nextID    int64

	posts   []postedMessage
	edits   []targetedMessage
	deletes []targetedMessage
}

func (r *recordingDiscord) Send(_ string, _ *discordgo.MessageSend) error { return nil }

func (r *recordingDiscord) SendWithID(channelID string, message *discordgo.MessageSend) (string, error) {
	r.Lock()
	defer r.Unlock()

	if !r.returnIDs {
		return "", nil
	}

	r.nextID++
	id := strconv.FormatInt(r.nextID, 10)
	r.posts = append(r.posts, postedMessage{channelID: channelID, message: message})

	return id, nil
}

func (r *recordingDiscord) EditMessage(channelID, messageID string, message *discordgo.MessageSend) error {
	r.Lock()
	defer r.Unlock()

	r.edits = append(r.edits, targetedMessage{channelID: channelID, messageID: messageID, message: message})

	return nil
}

func (r *recordingDiscord) DeleteMessage(channelID, messageID string) error {
	r.Lock()
	defer r.Unlock()

	r.deletes = append(r.deletes, targetedMessage{channelID: channelID, messageID: messageID})

	return nil
}

func (r *recordingDiscord) Start() error { return nil }
func (r *recordingDiscord) Close()       {}
func (r *recordingDiscord) Config() *discord.Config {
	return &discord.Config{}
}

func (r *recordingDiscord) MustRegisterCommandHandler(_ *discordgo.ApplicationCommand, _ discord.Handler) {
}
func (r *recordingDiscord) MustRegisterPrefixHandler(_ string, _ discord.Handler) {}
func (r *recordingDiscord) CreateRole(_ string) (string, error)                   { return "", nil }
func (r *recordingDiscord) Roles() ([]*discordgo.Role, error)                     { return nil, nil }

func messageHasHeading(message *discordgo.MessageSend, heading string) bool {
	if message == nil {
		return false
	}
	for _, component := range message.Components {
		if display, ok := component.(discordgo.TextDisplay); ok && strings.Contains(display.Content, heading) {
			return true
		}
	}

	return false
}

func TestSaveTracksDiscordMessage(t *testing.T) {
	t.Parallel()

	conn := &recordingDiscord{returnIDs: true}
	news.RegisterDiscordCommands(conn)

	svc := news.New(news.NewRepository(fixture.Database), conn, "news-channel")
	ctx := t.Context()

	title := stringutil.SecureRandomString(12)
	entry := &news.Article{Title: title, BodyMD: stringutil.SecureRandomString(40), IsPublished: true}
	require.NoError(t, svc.Save(ctx, entry))

	require.Len(t, conn.posts, 1)
	require.Equal(t, "news-channel", conn.posts[0].channelID)
	require.True(t, messageHasHeading(conn.posts[0].message, "News posted"))

	checked := news.Article{}
	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Equal(t, int64(1), checked.DiscordMessageID)

	// Editing the article must update the tracked message in place, not post
	// a duplicate.
	entry.Title = title + " (edited)"
	entry.BodyMD = stringutil.SecureRandomString(40)
	require.NoError(t, svc.Save(ctx, entry))

	require.Len(t, conn.posts, 1, "edit must not post a new discord message")
	require.Len(t, conn.edits, 1)
	require.Equal(t, "news-channel", conn.edits[0].channelID)
	require.Equal(t, "1", conn.edits[0].messageID)
	require.True(t, messageHasHeading(conn.edits[0].message, "News edited"))

	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Equal(t, int64(1), checked.DiscordMessageID, "tracked id must survive edits")

	// Deleting the article deletes the tracked discord message.
	require.NoError(t, svc.DropNewsArticle(ctx, entry.NewsID))

	require.Len(t, conn.deletes, 1)
	require.Equal(t, "news-channel", conn.deletes[0].channelID)
	require.Equal(t, "1", conn.deletes[0].messageID)
}

func TestSaveDeletesDiscordMessageWhenUnpublished(t *testing.T) {
	t.Parallel()

	conn := &recordingDiscord{returnIDs: true}
	svc := news.New(news.NewRepository(fixture.Database), conn, "news-channel")
	ctx := t.Context()

	title := stringutil.SecureRandomString(12)
	entry := &news.Article{Title: title, BodyMD: stringutil.SecureRandomString(40), IsPublished: true}
	require.NoError(t, svc.Save(ctx, entry))

	checked := news.Article{}
	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Equal(t, int64(1), checked.DiscordMessageID)

	// Unpublishing must delete the tracked discord message and clear the id.
	entry.IsPublished = false
	require.NoError(t, svc.Save(ctx, entry))

	require.Empty(t, conn.edits)
	require.Len(t, conn.deletes, 1)
	require.Equal(t, "news-channel", conn.deletes[0].channelID)
	require.Equal(t, "1", conn.deletes[0].messageID)

	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Zero(t, checked.DiscordMessageID, "tracked id must be cleared after a successful delete")

	// Saving the still-unpublished article again must not delete again.
	require.NoError(t, svc.Save(ctx, entry))
	require.Len(t, conn.deletes, 1)

	// Republishing posts a fresh message and tracks it.
	entry.IsPublished = true
	require.NoError(t, svc.Save(ctx, entry))

	require.Len(t, conn.posts, 2, "republish must post a fresh discord message")
	require.Empty(t, conn.edits)

	require.NoError(t, svc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Equal(t, int64(2), checked.DiscordMessageID)

	require.NoError(t, svc.DropNewsArticle(ctx, entry.NewsID))
	require.Len(t, conn.deletes, 2)
	require.Equal(t, "2", conn.deletes[1].messageID)
}

func TestSaveUnpublishedNewArticlePostsNothing(t *testing.T) {
	t.Parallel()

	conn := &recordingDiscord{returnIDs: true}
	svc := news.New(news.NewRepository(fixture.Database), conn, "news-channel")
	ctx := t.Context()

	entry := &news.Article{
		Title: stringutil.SecureRandomString(12), BodyMD: stringutil.SecureRandomString(40),
		IsPublished: false,
	}
	require.NoError(t, svc.Save(ctx, entry))

	require.Empty(t, conn.posts)
	require.Empty(t, conn.edits)
	require.Empty(t, conn.deletes)

	require.NoError(t, svc.DropNewsArticle(ctx, entry.NewsID))
}

func TestSavePostsNewDiscordMessageWhenUntracked(t *testing.T) {
	t.Parallel()

	news.RegisterDiscordCommands(nil)
	ctx := t.Context()

	// Simulate an article whose original post tracked no message id (the bot
	// was down when it posted): the first connection never returns an id.
	down := &recordingDiscord{}
	downSvc := news.New(news.NewRepository(fixture.Database), down, "news-channel")

	title := stringutil.SecureRandomString(12)
	entry := &news.Article{Title: title, BodyMD: stringutil.SecureRandomString(40), IsPublished: true}
	require.NoError(t, downSvc.Save(ctx, entry))

	checked := news.Article{}
	require.NoError(t, downSvc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Zero(t, checked.DiscordMessageID)

	// Once the bot is back, editing must post a fresh message and track it.
	restored := &recordingDiscord{returnIDs: true}
	restoredSvc := news.New(news.NewRepository(fixture.Database), restored, "news-channel")

	entry.Title = title + " (edited)"
	entry.BodyMD = stringutil.SecureRandomString(40)
	require.NoError(t, restoredSvc.Save(ctx, entry))

	require.Len(t, restored.posts, 1)
	require.True(t, messageHasHeading(restored.posts[0].message, "News edited"))
	require.Empty(t, restored.edits)

	require.NoError(t, restoredSvc.GetNewsByID(ctx, int(entry.NewsID), &checked))
	require.Equal(t, int64(1), checked.DiscordMessageID)

	require.NoError(t, restoredSvc.DropNewsArticle(ctx, entry.NewsID))
}
