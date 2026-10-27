package servers

import (
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/pkg/logparse"
)

// maxBufferedLines is the number of raw log lines retained in memory per server.
const maxBufferedLines = 10000

// BufferedLog is a single raw log line held in the in-memory per-server ring buffer.
type BufferedLog struct {
	Seq       uint64
	EventType logparse.EventType
	Raw       string
	CreatedOn time.Time
}

// serverRing is a fixed capacity ring buffer of log lines for a single server.
// Lines carry a monotonically increasing sequence number so a consumer can
// replay a snapshot and then continue on the live channel without losing or
// duplicating lines across the snapshot/live boundary.
type serverRing struct {
	lines []BufferedLog
	head  int
	size  int
	seq   uint64
}

func (r *serverRing) push(log BufferedLog) {
	r.lines[r.head] = log
	if r.size < maxBufferedLines {
		r.size++
	}
	r.head = (r.head + 1) % maxBufferedLines
}

// snapshot returns the buffered lines in chronological order.
func (r *serverRing) snapshot() []BufferedLog {
	if r.size < maxBufferedLines {
		return append([]BufferedLog(nil), r.lines[:r.size]...)
	}

	out := make([]BufferedLog, 0, maxBufferedLines)
	out = append(out, r.lines[r.head:]...)
	out = append(out, r.lines[:r.head]...)

	return out
}

// LogBuffer holds the most recent log lines per server instance and fans
// them out to subscribers. Push is safe for concurrent use and never blocks,
// so slow subscribers cannot stall the UDP log reader.
type LogBuffer struct {
	mu    sync.Mutex
	rings map[int32]*serverRing
	subs  map[int32][]chan<- BufferedLog
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		rings: map[int32]*serverRing{},
		subs:  map[int32][]chan<- BufferedLog{},
	}
}

// Push records a log line for the server and fans it out to any subscribers.
// Lines dropped from a full subscriber channel are discarded.
func (b *LogBuffer) Push(serverID int32, event logparse.ServerEvent) {
	b.mu.Lock()

	ring, found := b.rings[serverID]
	if !found {
		ring = &serverRing{lines: make([]BufferedLog, maxBufferedLines)}
		b.rings[serverID] = ring
	}

	ring.seq++
	log := BufferedLog{
		Seq:       ring.seq,
		EventType: event.EventType,
		Raw:       event.Raw,
		CreatedOn: event.CreatedOn,
	}
	ring.push(log)

	for _, sub := range b.subs[serverID] {
		select {
		case sub <- log:
		default:
			slog.Debug("Dropped log buffer event",
				slog.String("reason", "subscriber_queue_full"),
				slog.Int("server_id", int(serverID))) //nolint:gosec
		}
	}

	b.mu.Unlock()
}

// Snapshot returns a copy of the buffered lines for the server in chronological order.
func (b *LogBuffer) Snapshot(serverID int32) []BufferedLog {
	b.mu.Lock()
	defer b.mu.Unlock()

	ring, found := b.rings[serverID]
	if !found {
		return nil
	}

	return ring.snapshot()
}

// Consume registers a channel to receive the live log lines for the server.
// The channel must be buffered, events are dropped when it is full.
func (b *LogBuffer) Consume(serverID int32, ch chan<- BufferedLog) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs[serverID] = append(b.subs[serverID], ch)
}

// Unregister removes a channel previously registered with Consume.
func (b *LogBuffer) Unregister(subscriber chan<- BufferedLog) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for serverID, subs := range b.subs {
		out := subs[:0]
		for _, sub := range subs {
			if sub != subscriber {
				out = append(out, sub)
			}
		}

		if len(out) == 0 {
			delete(b.subs, serverID)
		} else {
			b.subs[serverID] = out
		}
	}
}
