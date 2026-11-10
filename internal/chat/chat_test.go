package chat_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sayEvent(msg string) logparse.ServerEvent {
	return logparse.ServerEvent{
		Results: logparse.Results{ //nolint:modernize // embedded field type is required in a keyed literal.
			EventType: logparse.Say,
			Event: logparse.SayEvt{
				TimeStamp:    logparse.TimeStamp{CreatedOn: time.Now()},
				SourcePlayer: logparse.SourcePlayer{Name: "Testy", PID: 1, SID: tests.GuestSID, Team: logparse.RED},
				Msg:          msg,
			},
		},
		Raw:        msg,
		CreatedOn:  time.Now(),
		ServerName: "test server",
	}
}

// TestChatWordFilterPipeline drives live UDP chat events through the broadcaster into the
// chat consumer and asserts the full word filter pipeline: match → warning → weight
// accumulation → exceed handler → persisted trigger count.
func TestChatWordFilterPipeline(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	events := broadcaster.New[logparse.EventType, logparse.ServerEvent]()

	cfg := &chat.Config{
		Enabled:      true,
		MaxWeight:    5,
		CheckTimeout: 5,
		MatchTimeout: 3600,
	}

	filters := chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database), notification.NullNotifier{}, cfg)
	fixture.CreateTestPerson(ctx, tests.OwnerSID)

	filter, errCreate := filters.Create(ctx, tests.OwnerSID, chat.Filter{
		Pattern:   "badword",
		IsEnabled: true,
		Action:    chat.FilterActionKick,
		Duration:  "1h",
		Weight:    3,
	})
	require.NoError(t, errCreate)
	require.NotZero(t, filter.FilterID)

	var (
		exceedCalls  atomic.Int32
		lastExceeded atomic.Bool
	)
	exceedHandler := func(_ context.Context, exceeded bool, _ chat.NewUserWarning) error {
		exceedCalls.Add(1)
		lastExceeded.Store(exceeded)

		return nil
	}

	chatUC := chat.New(chat.NewRepository(fixture.Database), cfg, filters,
		fixture.Persons, notification.NullNotifier{}, exceedHandler, "", tests.OwnerSID)

	go chatUC.Start(ctx, events)

	// Broadcaster.Emit is a no-op until the consumer registers, so give the
	// consumer goroutine a moment to start before emitting the first event.
	time.Sleep(time.Millisecond * 200) //nolint:mnd

	waitFor := func(cond func() bool, msg string) {
		require.Eventually(t, cond, time.Second*10, time.Millisecond*50, msg)
	}

	// First match: weight 3 is under max weight 5, so a warning is recorded without
	// exceeding the limit. This also proves the consumer does not block on the first match.
	events.Emit(logparse.Say, sayEvent("hello badword world"))
	waitFor(func() bool {
		state := chatUC.WarningState()

		return len(state[tests.GuestSID.String()]) == 1 && exceedCalls.Load() == 1
	}, "expected one warning and one exceed handler call")

	state := chatUC.WarningState()
	require.Len(t, state[tests.GuestSID.String()], 1)
	assert.Equal(t, int32(3), state[tests.GuestSID.String()][0].CurrentTotal)
	assert.False(t, lastExceeded.Load())

	// Second match: 3 + 3 = 6 exceeds max weight 5, so the exceed handler sees exceeded=true.
	events.Emit(logparse.Say, sayEvent("another badword"))
	waitFor(func() bool {
		st := chatUC.WarningState()

		return len(st[tests.GuestSID.String()]) == 2 && exceedCalls.Load() == 2
	}, "expected two warnings and two exceed handler calls")

	assert.True(t, lastExceeded.Load())
	state = chatUC.WarningState()
	assert.Equal(t, int32(6), state[tests.GuestSID.String()][1].CurrentTotal)

	// Trigger count is persisted for the filter.
	updated, errGet := filters.GetFilterByID(ctx, filter.FilterID)
	require.NoError(t, errGet)
	assert.EqualValues(t, 2, updated.TriggerCount)

	// No live person_messages or person_messages_filter rows are written by the live path;
	// matches are recorded when the demo import stores the message.
	var msgCount int
	require.NoError(t, fixture.Database.QueryRow(ctx, "SELECT count(*) FROM person_messages").Scan(&msgCount))
	assert.Zero(t, msgCount)

	var matchCount int
	require.NoError(t, fixture.Database.QueryRow(ctx, "SELECT count(*) FROM person_messages_filter").Scan(&matchCount))
	assert.Zero(t, matchCount)
}

// TestChatWordFilterDisabled ensures live chat events are not routed to the word filter
// system when the feature is disabled in config.
func TestChatWordFilterDisabled(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	events := broadcaster.New[logparse.EventType, logparse.ServerEvent]()

	cfg := &chat.Config{
		Enabled:      false,
		MaxWeight:    5,
		CheckTimeout: 5,
		MatchTimeout: 3600,
	}

	filters := chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database), notification.NullNotifier{}, cfg)
	fixture.CreateTestPerson(ctx, tests.OwnerSID)

	filter, errCreate := filters.Create(ctx, tests.OwnerSID, chat.Filter{
		Pattern:   "badword",
		IsEnabled: true,
		Action:    chat.FilterActionKick,
		Duration:  "1h",
		Weight:    3,
	})
	require.NoError(t, errCreate)

	var exceedCalls atomic.Int32
	exceedHandler := func(_ context.Context, _ bool, _ chat.NewUserWarning) error {
		exceedCalls.Add(1)

		return nil
	}

	chatUC := chat.New(chat.NewRepository(fixture.Database), cfg, filters,
		fixture.Persons, notification.NullNotifier{}, exceedHandler, "", tests.OwnerSID)

	go chatUC.Start(ctx, events)

	// Broadcaster.Emit is a no-op until the consumer registers, so give the
	// consumer goroutine a moment to start before emitting the event.
	time.Sleep(time.Millisecond * 200) //nolint:mnd

	events.Emit(logparse.Say, sayEvent("hello badword world"))

	// The emit only returns once the consumer received the event, so by the time the
	// event loop is free again the (nonexistent) warning side effects would already be done.
	time.Sleep(time.Second * 2) //nolint:mnd

	assert.Empty(t, chatUC.WarningState())
	assert.Zero(t, exceedCalls.Load())

	updated, errGet := filters.GetFilterByID(ctx, filter.FilterID)
	require.NoError(t, errGet)
	assert.Zero(t, updated.TriggerCount)
}

func TestChatBufferFanout(t *testing.T) {
	t.Parallel()

	buf := chat.NewBuffer()

	// Pushes must arrive in order with a monotonically increasing seq, including
	// pushes made before any subscriber exists.
	buf.Push(chat.BufferedChatMessage{ServerID: 1, Body: "one"})
	buf.Push(chat.BufferedChatMessage{ServerID: 1, Body: "two"})

	subA := make(chan chat.BufferedChatMessage, 4)
	subB := make(chan chat.BufferedChatMessage, 4)
	buf.Consume(subA)
	buf.Consume(subB)

	buf.Push(chat.BufferedChatMessage{ServerID: 2, Body: "three"})

	select {
	case msg := <-subA:
		assert.Equal(t, uint64(3), msg.Seq)
		assert.Equal(t, "three", msg.Body)
	default:
		t.Fatal("expected fan-out to receive message on subA")
	}

	select {
	case msg := <-subB:
		assert.Equal(t, uint64(3), msg.Seq)
	default:
		t.Fatal("expected fan-out to receive message on subB")
	}

	snapshot := buf.Snapshot()
	require.Len(t, snapshot, 3)
	assert.Equal(t, "one", snapshot[0].Body)
	assert.Equal(t, "two", snapshot[1].Body)
	assert.Equal(t, "three", snapshot[2].Body)

	// After unregistering a subscriber, only the remaining one gets messages.
	buf.Unregister(subA)
	buf.Push(chat.BufferedChatMessage{ServerID: 2, Body: "four"})

	select {
	case msg := <-subB:
		assert.Equal(t, "four", msg.Body)
	default:
		t.Fatal("expected fan-out to continue to subB after subA unregistered")
	}

	// A full subscriber channel must not block the push path; extra messages are dropped.
	subC := make(chan chat.BufferedChatMessage, 1)
	buf.Consume(subC)
	buf.Unregister(subB)
	buf.Push(chat.BufferedChatMessage{ServerID: 2, Body: "five"})
	buf.Push(chat.BufferedChatMessage{ServerID: 2, Body: "six"}) // dropped, subC is full

	select {
	case msg := <-subC:
		assert.Equal(t, "five", msg.Body)
	default:
		t.Fatal("expected subC to receive a message")
	}

	assert.Empty(t, subC)

	// Pushing and snapshotting happen concurrently in production, so exercise
	// the locked paths under -race. The message dropped from the subscriber was
	// still retained in the ring, hence 6 snapshots.
	for range 10 {
		assert.Len(t, buf.Snapshot(), 6)
	}
}

func TestWordFilterCheckRespectsEnabled(t *testing.T) {
	t.Parallel()

	filters := chat.NewWordFilters(chat.NewWordFilterRepository(nil), notification.NullNotifier{}, &chat.Config{})

	filters.Add(chat.Filter{Pattern: "slur", IsEnabled: false})
	assert.Empty(t, filters.Check("a slur here"))

	filters.Add(chat.Filter{Pattern: "slur", IsEnabled: true})
	matches := filters.Check("a slur here")
	require.Len(t, matches, 1)
	assert.Equal(t, "slur", matches[0].Pattern)
}
