package news_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
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

	svc := news.New(news.NewRepository(fixture.Database), notification.NullNotifier{}, "")
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

type capturingNotifier struct {
	sync.Mutex

	payloads []notification.Payload
}

func (c *capturingNotifier) Send(payload notification.Payload) {
	c.Lock()
	defer c.Unlock()

	c.payloads = append(c.payloads, payload)
}

func (c *capturingNotifier) list() []notification.Payload {
	c.Lock()
	defer c.Unlock()

	out := make([]notification.Payload, len(c.payloads))
	copy(out, c.payloads)

	return out
}

func containsHeading(payloads []notification.Payload, heading string) bool {
	for _, payload := range payloads {
		if payload.MessageSend == nil {
			continue
		}

		for _, component := range payload.MessageSend.Components {
			if display, ok := component.(discordgo.TextDisplay); ok && strings.Contains(display.Content, heading) {
				return true
			}
		}
	}

	return false
}

func TestSaveNotificationHeadings(t *testing.T) {
	t.Parallel()

	news.RegisterDiscordCommands(discord.NewDiscard(&discord.Config{}))

	notifier := &capturingNotifier{}
	svc := news.New(news.NewRepository(fixture.Database), notifier, "test-channel")
	ctx := t.Context()

	entry := &news.Article{Title: "Notification test", BodyMD: "Body long enough for notification", IsPublished: true}
	require.NoError(t, svc.Save(ctx, entry))

	require.Eventually(t, func() bool {
		return containsHeading(notifier.list(), "News posted")
	}, time.Second*2, time.Millisecond*10)

	entry.Title = "Notification test (edited)"
	require.NoError(t, svc.Save(ctx, entry))

	require.Eventually(t, func() bool {
		return containsHeading(notifier.list(), "News edited")
	}, time.Second*2, time.Millisecond*10)

	require.NoError(t, svc.DropNewsArticle(ctx, entry.NewsID))
}
