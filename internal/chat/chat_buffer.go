package chat

import (
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

// maxBufferedChat is the total number of chat messages retained in memory.
const maxBufferedChat = 1000

// BufferedChatMessage is a single live chat message held in the in-memory ring buffer.
type BufferedChatMessage struct {
	Seq         uint64
	SteamID     steamid.SteamID
	AvatarHash  string
	PersonaName string
	ServerID    int32
	ServerName  string
	Body        string
	Team        bool
	CreatedOn   time.Time
}

// Buffer retains the most recent live chat messages across all servers and
// fans them out to streaming subscribers. Push is safe for concurrent use and
// never blocks, so slow subscribers cannot stall the log reader.
type Buffer struct {
	mu   sync.Mutex
	ring [maxBufferedChat]BufferedChatMessage
	head int
	size int
	seq  uint64
	subs []chan<- BufferedChatMessage
}

func NewBuffer() *Buffer {
	return &Buffer{}
}

// Push records a chat message and fans it out to any subscribers. Messages
// dropped from a full subscriber channel are discarded.
func (b *Buffer) Push(msg BufferedChatMessage) {
	b.mu.Lock()

	b.seq++
	msg.Seq = b.seq
	b.ring[b.head] = msg
	b.head = (b.head + 1) % maxBufferedChat
	if b.size < maxBufferedChat {
		b.size++
	}

	for _, sub := range b.subs {
		select {
		case sub <- msg:
		default:
			slog.Debug("Dropped chat buffer event",
				slog.String("reason", "subscriber_queue_full"),
				slog.Int("server_id", int(msg.ServerID))) //nolint:gosec
		}
	}

	b.mu.Unlock()
}

// Snapshot returns a copy of the buffered messages in chronological order.
func (b *Buffer) Snapshot() []BufferedChatMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size < maxBufferedChat {
		return append([]BufferedChatMessage(nil), b.ring[:b.size]...)
	}

	out := make([]BufferedChatMessage, 0, maxBufferedChat)
	out = append(out, b.ring[b.head:]...)
	out = append(out, b.ring[:b.head]...)

	return out
}

// Consume registers a channel to receive live chat messages. The channel must
// be buffered, messages are dropped when it is full.
func (b *Buffer) Consume(ch chan<- BufferedChatMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs = append(b.subs, ch)
}

// Unregister removes a channel previously registered with Consume.
func (b *Buffer) Unregister(sub chan<- BufferedChatMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := b.subs[:0]
	for _, s := range b.subs {
		if s != sub {
			out = append(out, s)
		}
	}

	b.subs = out
}
