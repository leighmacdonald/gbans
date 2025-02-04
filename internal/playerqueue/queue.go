package playerqueue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

var (
	ErrUnexpectedMessage = errors.New("unexpected message")
	errMessageID         = errors.New("failed to create message id")
	ErrQueueIO           = errors.New("failed to read / write from connection")
	ErrQueueParseMessage = errors.New("failed to parse message")
	ErrBadInput          = errors.New("bad user input")
)

// TODO Track a users desired minimum size for them to be counted towards play.
// Show users queue size for both at their min levels and without.

type Client struct {
	user         domain.UserProfile
	conn         *websocket.Conn
	messageQueue chan Msg
	lastPing     time.Time
}

func (c *Client) startSender(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.messageQueue:
			if errWrite := c.conn.WriteJSON(msg); errWrite != nil {
				slog.Error("Failed to send message to client", log.ErrAttr(errWrite))
			}
		}
	}
}

func (c *Client) close() {
	slog.Debug("Closing client connection", slog.String("addr", c.conn.RemoteAddr().String()))
	if errClose := c.conn.Close(); errClose != nil {
		slog.Warn("Error closing client connection", log.ErrAttr(errClose))
	}
}

type Queue struct {
	chatLogHistorySize int
	minQueueSize       int
	serverQueues       []*ServerQueueState
	clients            []*Client
	chatLogs           []domain.Message
	servers            domain.ServersUsecase
	serverState        domain.StateUsecase
	mu                 *sync.RWMutex
}

func NewServerQueue(chatLogHistorySize int, minQueueSize int, servers domain.ServersUsecase,
	serverState domain.StateUsecase, chatlogs []domain.Message,
) *Queue {
	return &Queue{
		minQueueSize:       minQueueSize,
		clients:            []*Client{},
		chatLogs:           chatlogs,
		serverQueues:       []*ServerQueueState{},
		mu:                 &sync.RWMutex{},
		chatLogHistorySize: chatLogHistorySize,
		serverState:        serverState,
		servers:            servers,
	}
}

func (q *Queue) Start(ctx context.Context) {
	cleanupTicker := time.NewTicker(time.Second * 30)
	queueQueck := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-cleanupTicker.C:
			q.removeZombies()
		case <-queueQueck.C:
			if err := q.checkQueueCompat(ctx); err != nil {
				slog.Error("Failed to check queue compatibility", log.ErrAttr(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *Queue) removeZombies() {
	q.mu.Lock()
	defer q.mu.Unlock()

	var valid []*Client
	for _, client := range q.clients {
		if time.Since(client.lastPing) > time.Minute {
			q.removeFromQueues(client)
			client.close()
			slog.Debug("Removing zombie client", slog.String("addr", client.conn.RemoteAddr().String()))
		} else {
			valid = append(valid, client)
		}
	}

	q.clients = valid
}

func (q *Queue) syncServerIDs(ctx context.Context) {
	servers, _, errServers := q.servers.Servers(ctx, domain.ServerQueryFilter{
		QueryFilter:     domain.QueryFilter{},
		IncludeDisabled: false,
	})

	if errServers != nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	//goland:noinspection GoPreferNilSlice
	valid := []*ServerQueueState{}

	for _, srv := range servers {
		found := false
		for _, serverQueue := range q.serverQueues {
			if serverQueue.ServerID == srv.ServerID {
				// Keep any existing queue states that remain valid.
				valid = append(valid, serverQueue)
				found = true

				break
			}
		}

		if !found {
			// Create new entries for missing keys.
			valid = append(valid, &ServerQueueState{
				ServerID: srv.ServerID,
				Members:  []ClientQueueState{},
			})
		}
	}

	q.serverQueues = valid
}

func (q *Queue) updateClientStates(fullUpdate bool) {
	update := clientStatePayload{}

	q.mu.RLock()
	defer q.mu.RUnlock()

	//goland:noinspection GoPreferNilSlice
	updateMap := []ServerQueueState{}
	for _, value := range q.serverQueues {
		updateMap = append(updateMap, *value)
	}

	update.UpdateServers = true
	update.Servers = updateMap

	if fullUpdate {
		//goland:noinspection GoPreferNilSlice
		players := []member{}
		for _, client := range q.clients {
			players = append(players, member{
				Name:    client.user.GetName(),
				SteamID: client.user.SteamID.String(),
				Hash:    client.user.Avatarhash,
			})
		}
		update.UpdateUsers = true
		update.Users = players
	}

	q.broadcast(Msg{Op: StateUpdate, Payload: update})
}

func (q *Queue) LeaveQueue(client *Client, servers []int) error {
	changed := false

	q.mu.Lock()
	for _, serverID := range servers {
		for _, srv := range q.serverQueues {
			if srv.ServerID != serverID {
				continue
			}

			//goland:noinspection GoPreferNilSlice
			valid := []ClientQueueState{}
			for _, mem := range srv.Members {
				if mem.SteamID != client.user.SteamID {
					valid = append(valid, mem)
				} else {
					changed = true
				}
			}

			srv.Members = valid
		}
	}
	q.mu.Unlock()

	if changed {
		q.updateClientStates(false)
	}

	return nil
}

func (q *Queue) JoinQueue(ctx context.Context, client *Client, servers []int) error {
	changed := false
	q.mu.Lock()

	for _, serverID := range servers {
		for _, srv := range q.serverQueues {
			if srv.ServerID != serverID {
				continue
			}

			found := false
			for _, mem := range srv.Members {
				if mem.SteamID == client.user.SteamID {
					found = true
				}
			}

			if !found {
				srv.Members = append(srv.Members, ClientQueueState{SteamID: client.user.SteamID})
				changed = true
			}

			break
		}
	}

	q.mu.Unlock()

	if changed {
		q.updateClientStates(false)

		return q.checkQueueCompat(ctx)
	}

	return nil
}

// Connect adds the user to the swarm. If a user exists with the same steamid exists, it will be replaced with
// the new connection.
func (q *Queue) Connect(ctx context.Context, user domain.UserProfile, conn *websocket.Conn) *Client {
	// Sync valid servers each time a client connects.
	q.syncServerIDs(ctx)

	q.mu.Lock()
	defer q.mu.Unlock()

	client := &Client{
		user:         user,
		conn:         conn,
		messageQueue: make(chan Msg, 2),
		lastPing:     time.Time{},
	}

	for i := range q.clients {
		if q.clients[i].user.SteamID == user.SteamID {
			q.clients[i].close()

			break
		}
	}

	q.clients = append(q.clients, client)

	go client.startSender(ctx)
	go q.sendClientChatHistory(client)
	go q.updateClientStates(true)

	return client
}

func (q *Queue) checkQueueCompat(ctx context.Context) error {
	var serverID int

	q.mu.Lock()
	for _, serverQueue := range q.serverQueues {
		if len(serverQueue.Members) < q.minQueueSize {
			continue
		}

		state, found := q.serverState.ByServerID(serverQueue.ServerID)
		if !found {
			continue
		}

		if state.MaxPlayers-state.PlayerCount-len(serverQueue.Members) < 0 {
			continue
		}

		serverID = serverQueue.ServerID

		break
	}
	q.mu.Unlock()

	if serverID > 0 {
		return q.initiateGame(ctx, serverID)
	}

	return nil
}

func (q *Queue) initiateGame(ctx context.Context, serverID int) error {
	srv, errServer := q.servers.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	q.mu.RLock()
	var queuedClients []*Client

	for _, serverQueue := range q.serverQueues {
		if serverQueue.ServerID != serverID {
			continue
		}

		// Find the queued users via their matching steamid
		for _, client := range q.clients {
			for _, c := range serverQueue.Members {
				if client.user.SteamID == c.SteamID {
					queuedClients = append(queuedClients, client)

					break
				}
			}
		}
	}

	ipAddr, errIP := srv.IP(ctx)
	if errIP != nil {
		q.mu.RUnlock()

		return errors.Join(errIP, domain.ErrResolveIP)
	}

	startPayload := gameStartPayload{
		Server: server{
			Name:           srv.Name,
			ShortName:      srv.ShortName,
			CC:             srv.CC,
			ConnectURL:     fmt.Sprintf("steam://connect/%s:%d", ipAddr.String(), srv.Port),
			ConnectCommand: fmt.Sprintf("connect %s:%d", srv.Address, srv.Port),
		},
	}

	for _, target := range queuedClients {
		startPayload.Users = append(startPayload.Users, member{
			Name:    target.user.Name,
			SteamID: target.user.SteamID.String(),
			Hash:    target.user.Avatarhash,
		})
	}

	q.broadcast(Msg{Op: StartGame, Payload: startPayload}, queuedClients...)
	q.mu.RUnlock()

	q.mu.Lock()
	for _, client := range queuedClients {
		q.removeFromQueues(client)
	}
	q.mu.Unlock()

	q.updateClientStates(false)

	return nil
}

func (q *Queue) findMessages(steamID steamid.SteamID, limit int) []domain.Message {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var messages []domain.Message
	for i := len(q.chatLogs); i > 0; i-- {
		if q.chatLogs[i].SteamID == steamID.String() {
			messages = append(messages, q.chatLogs[i])
			if len(messages) == limit {
				return messages
			}
		}
	}

	return messages
}

func (q *Queue) purgeMessages(ids ...uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove the purged messages from the local cache.
	var valid []domain.Message
	for _, existing := range q.chatLogs {
		if !slices.Contains(ids, existing.MessageID) {
			valid = append(valid, existing)
		}
	}
	q.chatLogs = valid

	q.broadcast(Msg{Op: Purge, Payload: purgePayload{MessageIDs: ids}})
}

func (q *Queue) removeFromQueues(client *Client) {
	for _, srv := range q.serverQueues {
		var valid []ClientQueueState

		for _, mem := range srv.Members {
			if mem.SteamID != client.user.SteamID {
				valid = append(valid, mem)
			}
		}

		srv.Members = valid
	}
}

func (q *Queue) Disconnect(client *Client) {
	q.mu.RLock()

	var serverIDs []int //nolint:prealloc
	for server := range q.serverQueues {
		serverIDs = append(serverIDs, server)
	}

	q.mu.RUnlock()

	if err := q.LeaveQueue(client, serverIDs); err != nil {
		slog.Warn("Error leaving server queue", log.ErrAttr(err))
	}

	q.mu.Lock()
	var valid []*Client //nolint:prealloc
	for _, existing := range q.clients {
		if existing.user.SteamID == client.user.SteamID {
			client.close()

			continue
		}

		valid = append(valid, existing)
	}

	q.clients = valid
	q.mu.Unlock()

	q.updateClientStates(true)
}

func (q *Queue) Message(message domain.Message, user domain.UserProfile) (domain.Message, error) {
	message.BodyMD = sanitizeUserMessage(message.BodyMD)
	if len(message.BodyMD) == 0 {
		return message, ErrBadInput
	}

	id, errID := uuid.NewV7()
	if errID != nil {
		return message, errors.Join(errID, errMessageID)
	}

	message.MessageID = id
	message.CreatedOn = time.Now()
	message.Avatarhash = user.Avatarhash
	message.Personaname = user.GetName()
	message.SteamID = user.SteamID.String()

	q.mu.Lock()
	q.chatLogs = append(q.chatLogs, message)
	if len(q.chatLogs) > q.chatLogHistorySize {
		q.chatLogs = q.chatLogs[1:]
	}
	q.mu.Unlock()

	q.mu.RLock()
	q.broadcast(Msg{
		Op:      MessageRecv,
		Payload: message,
	})
	q.mu.RUnlock()

	return message, nil
}

func (q *Queue) broadcast(payload Msg, targetClients ...*Client) {
	if len(targetClients) == 0 {
		targetClients = q.clients
	}

	for _, client := range targetClients {
		slog.Debug("Sending message to client", slog.Int("op", int(payload.Op)),
			slog.String("client", client.conn.RemoteAddr().String()))
		client.messageQueue <- payload
	}
}

func sanitizeUserMessage(msg string) string {
	s := removeNonPrintable(strings.TrimSpace(msg))
	s = stringutil.SanitizeUGC(s)
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

func (q *Queue) Ping(client *Client) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	client.messageQueue <- Msg{
		Op:      Pong,
		Payload: pongPayload{CreatedOn: time.Now()},
	}
}

func (q *Queue) sendClientChatHistory(client *Client) {
	q.mu.RLock()
	msgs := make([]Msg, len(q.chatLogs))
	for i, cl := range q.chatLogs {
		msgs[i] = Msg{
			Op:      MessageRecv,
			Payload: cl,
		}
	}
	q.mu.RUnlock()

	for _, msg := range msgs {
		client.messageQueue <- msg
	}
}
