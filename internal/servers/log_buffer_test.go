package servers_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func makeEvent(serverID int32, n int) logparse.ServerEvent {
	return logparse.ServerEvent{
		Results:   logparse.Results{EventType: logparse.Say}, //nolint:modernize // embedded field type is required in a keyed literal.
		CreatedOn: time.Now(),
		Raw:       fmt.Sprintf("L 01/01/2026 - 00:00:00: [CHAT] Test <1><><> says \"line %d\"", n),
		ServerID:  serverID,
	}
}

func drainLogChannel(ch chan servers.BufferedLog) []servers.BufferedLog {
	var out []servers.BufferedLog
	for {
		select {
		case log := <-ch:
			out = append(out, log)
		default:
			return out
		}
	}
}

func TestLogBufferPushSnapshot(t *testing.T) {
	buffer := servers.NewLogBuffer()

	for n := 1; n <= 25; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	snap := buffer.Snapshot(1)
	require.Len(t, snap, 25)
	for n, log := range snap {
		require.Equal(t, uint64(n+1), log.Seq)
		require.Contains(t, log.Raw, fmt.Sprintf("line %d", n+1))
	}

	// Unknown servers have an empty buffer.
	require.Empty(t, buffer.Snapshot(99))
}

func TestLogBufferRingEviction(t *testing.T) {
	buffer := servers.NewLogBuffer()

	const total = 10001
	for n := 1; n <= total; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	snap := buffer.Snapshot(1)
	require.Len(t, snap, 10000)

	// The first line was evicted, ordering and seqs are intact.
	require.Contains(t, snap[0].Raw, "line 2")
	require.NotContains(t, snap[0].Raw, "line 1")
	require.Contains(t, snap[len(snap)-1].Raw, "line 10001")

	for i, log := range snap {
		require.Equal(t, uint64(i+2), log.Seq)
	}
}

func TestLogBufferServerIsolation(t *testing.T) {
	buffer := servers.NewLogBuffer()

	for n := 1; n <= 3; n++ {
		buffer.Push(1, makeEvent(1, n))
	}
	for n := 1; n <= 5; n++ {
		buffer.Push(2, makeEvent(2, n))
	}

	require.Len(t, buffer.Snapshot(1), 3)
	require.Len(t, buffer.Snapshot(2), 5)

	// Sequence numbers are per-server.
	snap2 := buffer.Snapshot(2)
	require.Equal(t, uint64(1), snap2[0].Seq)
	require.Equal(t, uint64(5), snap2[len(snap2)-1].Seq)
}

// TestLogBufferSnapshotLiveBoundary covers the dedup contract used by the
// streaming handler: subscribe, snapshot, then consume the live channel.
// Lines pushed between the subscribe and the snapshot appear in both, so the
// consumer must skip live lines at or below the snapshot watermark. The
// deduped union must contain every line exactly once.
func TestLogBufferSnapshotLiveBoundary(t *testing.T) {
	buffer := servers.NewLogBuffer()

	for n := 1; n <= 5; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	sub := make(chan servers.BufferedLog, 100)
	buffer.Consume(1, sub)

	for n := 6; n <= 10; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	snap := buffer.Snapshot(1)
	require.Len(t, snap, 10)
	watermark := snap[len(snap)-1].Seq

	for n := 11; n <= 15; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	live := drainLogChannel(sub)
	require.NotEmpty(t, live)

	seen := make(map[uint64]int)
	for _, log := range snap {
		seen[log.Seq]++
	}
	duped := 0
	for _, log := range live {
		if log.Seq <= watermark {
			// Duplicates of the snapshot window, skipped by the handler.
			duped++

			continue
		}
		seen[log.Seq]++
	}

	// Lines 6-10 were pushed between subscribe and snapshot, so they must be
	// the duplicates.
	require.Equal(t, 5, duped)

	require.Len(t, seen, 15)
	for seq := uint64(1); seq <= 15; seq++ {
		require.Equalf(t, 1, seen[seq], "line %d delivered %d times", seq, seen[seq])
	}
}

func TestLogBufferUnregister(t *testing.T) {
	buffer := servers.NewLogBuffer()

	event := makeEvent(1, 1)
	sub := make(chan servers.BufferedLog, 100)
	buffer.Consume(1, sub)
	buffer.Push(1, event)
	buffer.Unregister(sub)

	buffer.Push(1, makeEvent(1, 2))

	require.Equal(t, []servers.BufferedLog{
		{Seq: 1, EventType: logparse.Say, Raw: event.Raw, CreatedOn: event.CreatedOn},
	}, drainLogChannel(sub))
	require.Len(t, buffer.Snapshot(1), 2)
}

func TestLogBufferDropsOnFullSubscriber(t *testing.T) {
	buffer := servers.NewLogBuffer()

	sub := make(chan servers.BufferedLog, 1)
	buffer.Consume(1, sub)
	defer buffer.Unregister(sub)

	// No reader, capacity 1: pushes must not block or panic.
	for n := 1; n <= 10; n++ {
		buffer.Push(1, makeEvent(1, n))
	}

	require.Len(t, drainLogChannel(sub), 1)
	require.Len(t, buffer.Snapshot(1), 10)
}

func TestLogBufferConcurrent(t *testing.T) {
	buffer := servers.NewLogBuffer()

	const (
		goroutines = 4
		lines      = 2500
	)

	var pushers sync.WaitGroup
	for range goroutines {
		pushers.Go(func() {
			for n := 1; n <= lines; n++ {
				buffer.Push(1, makeEvent(1, n))
			}
		})
	}

	// Snapshot concurrently while the pushers are writing.
	snapshotStop := make(chan struct{})
	var snapshotWG sync.WaitGroup
	snapshotWG.Go(func() {
		for {
			select {
			case <-snapshotStop:
				return
			default:
				snap := buffer.Snapshot(1)
				for i := 1; i < len(snap); i++ {
					if snap[i].Seq <= snap[i-1].Seq {
						t.Errorf("snapshot not in seq order at index %d", i)

						return
					}
				}
				time.Sleep(time.Millisecond)
			}
		}
	})

	pushers.Wait()
	close(snapshotStop)
	snapshotWG.Wait()

	snap := buffer.Snapshot(1)
	require.Len(t, snap, 10000)
	for i := 1; i < len(snap); i++ {
		require.Greater(t, snap[i].Seq, snap[i-1].Seq)
	}
}
