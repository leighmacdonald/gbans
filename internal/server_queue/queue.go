package server_queue

import (
	"context"
	"errors"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"
)

var (
	ErrQueueUnknownUser   = errors.New("unknown queue user")
	ErrQueueUnknownServer = errors.New("unknown queue server")
	ErrUnexpectedMessage  = errors.New("unexpected message")
	ErrQueueIO            = errors.New("failed to read / write from connection")
	ErrQueueParseMessage  = errors.New("failed to parse message")
)

// TODO Track a users desired minimum size for them to be counted towards play.
// Show users queue size for both at their min levels and without.

type ClientConn struct {
	user                 domain.UserProfile
	conn                 *websocket.Conn
	outboundMessageQueue chan domain.ServerQueuePayloadOutbound
	hasPinged            bool
	lastPing             time.Time
}

func (c *ClientConn) startSender(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.outboundMessageQueue:
			if errWrite := c.conn.WriteJSON(msg); errWrite != nil {
				slog.Error("Failed to send message to client", log.ErrAttr(errWrite))
			}
		}
	}
}

func (c *ClientConn) close() {
	slog.Debug("Closing client connection", slog.String("addr", c.conn.RemoteAddr().String()))
	if errClose := c.conn.Close(); errClose != nil {
		slog.Warn("Error closing client connection", log.ErrAttr(errClose))
	}
}

type ServerQueue struct {
	servers  map[int][]steamid.SteamID
	mu       *sync.RWMutex
	clients  []*ClientConn
	chatLogs []domain.ServerQueueMessage
}

func NewServerQueue() *ServerQueue {
	return &ServerQueue{
		clients:  make([]*ClientConn, 0),
		chatLogs: make([]domain.ServerQueueMessage, 0),
		servers:  map[int][]steamid.SteamID{},
		mu:       &sync.RWMutex{},
	}
}

func (q *ServerQueue) JoinQueue(client *ClientConn, servers []int) error {
	changed := false
	q.mu.Lock()

	for _, serverID := range servers {
		members, exists := q.servers[serverID]
		if !exists {
			q.mu.Unlock()

			return ErrQueueUnknownServer
		}

		found := false
		for _, member := range members {
			if member == client.user.SteamID {
				found = true
			}
		}

		if !found {
			q.servers[serverID] = append(q.servers[serverID], client.user.SteamID)
			changed = true
		}
	}

	q.mu.Unlock()

	if changed {
		q.broadcast(domain.ServerQueuePayloadOutbound{Op: domain.StateUpdate, Payload: q.servers})
	}

	return nil
}

func (q *ServerQueue) ConnectClient(ctx context.Context, user domain.UserProfile, conn *websocket.Conn) *ClientConn {
	q.mu.Lock()
	defer q.mu.Unlock()

	client := &ClientConn{
		user:                 user,
		conn:                 conn,
		outboundMessageQueue: make(chan domain.ServerQueuePayloadOutbound, 2),
		hasPinged:            false,
		lastPing:             time.Time{},
	}

	go client.startSender(ctx)

	for i := range q.clients {
		if q.clients[i].user.SteamID == user.SteamID {
			q.clients[i].close()
			q.clients[i] = client

			return client
		}
	}

	q.clients = append(q.clients, client)

	go q.sendClientChatHistory(client)

	return client
}

func (q *ServerQueue) DisconnectClient(leaver *ClientConn) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var valid []*ClientConn
	for _, client := range q.clients {
		if client.user.SteamID == leaver.user.SteamID {
			client.close()

			continue
		}
		valid = append(valid, client)
	}

	q.clients = valid
}

func (q *ServerQueue) Start(serverID int) {

}

func (q *ServerQueue) Message(p domain.ServerQueueMessage, user domain.UserProfile) error {
	p.BodyMD = sanitizeUserMessage(p.BodyMD)
	if len(p.BodyMD) == 0 {
		return nil
	}
	id, errID := uuid.NewV7()
	if errID != nil {
		return errID
	}

	p.ID = id
	p.CreatedOn = time.Now()
	p.Avatarhash = user.Avatarhash
	p.Personaname = user.GetName()
	p.SteamID = user.SteamID.String()

	q.mu.Lock()
	q.chatLogs = append(q.chatLogs, p)
	if len(q.chatLogs) > 5 {
		q.chatLogs = q.chatLogs[1:]
	}
	q.mu.Unlock()

	q.broadcast(domain.ServerQueuePayloadOutbound{
		Op:      domain.MessageRecv,
		Payload: p,
	})

	return nil
}

func (q *ServerQueue) broadcast(payload domain.ServerQueuePayloadOutbound) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, client := range q.clients {
		slog.Debug("sending message to client", slog.String("client", client.conn.RemoteAddr().String()))
		client.outboundMessageQueue <- payload
	}
}

func sanitizeUserMessage(msg string) string {
	s := removeNonPrintable(strings.TrimSpace(msg))
	// TODO 1984
	return s
}

func removeNonPrintable(input string) string {
	out := strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) && unicode.IsPrint(r) || r == ' ' {
			return r
		}
		return -1
	}, input)

	return out
}

func (q *ServerQueue) removeClient(c *ClientConn) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var valid []*ClientConn
	for _, client := range q.clients {
		if client != c {
			valid = append(valid, client)
		}
	}

	q.clients = valid
}

func (q *ServerQueue) Ping(client *ClientConn) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	client.outboundMessageQueue <- domain.ServerQueuePayloadOutbound{
		Op:      domain.Pong,
		Payload: domain.PongPayload{CreatedOn: time.Now()},
	}
}

func (q *ServerQueue) sendClientChatHistory(client *ClientConn) {
	q.mu.RLock()

	var msgs []domain.ServerQueuePayloadOutbound
	for _, cl := range q.chatLogs {
		msgs = append(msgs, domain.ServerQueuePayloadOutbound{
			Op:      domain.MessageRecv,
			Payload: cl,
		})
	}

	q.mu.RUnlock()

	for _, msg := range msgs {
		client.outboundMessageQueue <- msg
	}
}
