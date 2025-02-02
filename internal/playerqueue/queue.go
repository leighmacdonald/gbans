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
)

var (
	errQueueUnknownServer  = errors.New("unknown queue server")
	ErrUnexpectedMessage   = errors.New("unexpected message")
	ErrQueueMissingEntries = errors.New("failed to find all queue members")
	errMessageID           = errors.New("failed to create message id")
	ErrQueueIO             = errors.New("failed to read / write from connection")
	ErrQueueParseMessage   = errors.New("failed to parse message")
	ErrBadInput            = errors.New("bad user input")
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

type ServerQueue struct {
	chatLogHistorySize int
	state              map[int][]steamid.SteamID
	mu                 *sync.RWMutex
	clients            []*Client
	chatLogs           []domain.Message
	servers            domain.ServersUsecase
}

func NewServerQueue(chatLogHistorySize int, servers domain.ServersUsecase) *ServerQueue {
	return &ServerQueue{
		clients:            make([]*Client, 0),
		chatLogs:           []domain.Message{},
		state:              map[int][]steamid.SteamID{},
		mu:                 &sync.RWMutex{},
		chatLogHistorySize: chatLogHistorySize,
		servers:            servers,
	}
}

func (q *ServerQueue) syncServerIDs(ctx context.Context) {
	servers, _, errServers := q.servers.Servers(ctx, domain.ServerQueryFilter{
		QueryFilter:     domain.QueryFilter{},
		IncludeDisabled: false,
	})

	if errServers != nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	valid := map[int][]steamid.SteamID{}

	for _, validID := range servers {
		if _, found := q.state[validID.ServerID]; found {
			// Keep any existing queue states that remain valid.
			valid[validID.ServerID] = q.state[validID.ServerID]
		} else {
			// Create new entries for missing keys.
			valid[validID.ServerID] = []steamid.SteamID{}
		}
	}

	q.state = valid
}

func (q *ServerQueue) updateClientStates(fullUpdate bool) {
	update := clientStatePayload{}

	q.mu.RLock()
	defer q.mu.RUnlock()

	//goland:noinspection GoPreferNilSlice
	updateMap := []ServerQueueState{}
	for key, value := range q.state {
		updateMap = append(updateMap, ServerQueueState{
			ServerID: key,
			Members:  value,
		})
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

func (q *ServerQueue) LeaveQueue(client *Client, servers []int) error {
	q.mu.Lock()

	changed := false
	for _, serverID := range servers {
		members, ok := q.state[serverID]
		if !ok {
			q.mu.Unlock()

			return errQueueUnknownServer
		}

		//goland:noinspection GoPreferNilSlice
		valid := []steamid.SteamID{}
		for _, member := range members {
			if member != client.user.SteamID {
				valid = append(valid, member)
			} else {
				changed = true
			}
		}

		q.state[serverID] = valid
	}
	q.mu.Unlock()

	if changed {
		q.updateClientStates(false)
	}

	return nil
}

func (q *ServerQueue) JoinQueue(ctx context.Context, client *Client, servers []int) error {
	changed := false
	q.mu.Lock()

	for _, serverID := range servers {
		members, exists := q.state[serverID]
		if !exists {
			q.mu.Unlock()

			return errQueueUnknownServer
		}

		found := false
		for _, mem := range members {
			if mem == client.user.SteamID {
				found = true
			}
		}

		if !found {
			q.state[serverID] = append(q.state[serverID], client.user.SteamID)
			changed = true
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
func (q *ServerQueue) Connect(ctx context.Context, user domain.UserProfile, conn *websocket.Conn) *Client {
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

func (q *ServerQueue) checkQueueCompat(ctx context.Context) error {
	const minSize = 2

	q.mu.RLock()
	defer q.mu.RUnlock()

	for serverID, members := range q.state {
		if len(members) < minSize {
			continue
		}

		if err := q.initiateGame(ctx, serverID); err != nil {
			return err
		}

		break
	}

	return nil
}

type server struct {
	Name           string `json:"name"`
	ShortName      string `json:"short_name"`
	CC             string `json:"cc"`
	ConnectURL     string `json:"connect_url"`
	ConnectCommand string `json:"connect_command"`
}

type gameStartPayload struct {
	Users  []member `json:"users"`
	Server server   `json:"server"`
}

func (q *ServerQueue) initiateGame(ctx context.Context, serverID int) error {
	srv, errServer := q.servers.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	var sendTargets []*Client

	for _, steamID := range q.state[serverID] {
		for _, client := range q.clients {
			if client.user.SteamID == steamID {
				sendTargets = append(sendTargets, client)

				break
			}
		}
	}

	if len(sendTargets) != len(q.state[serverID]) {
		return ErrQueueMissingEntries
	}

	startPayload := gameStartPayload{
		Server: server{
			Name:           srv.Name,
			ShortName:      srv.ShortName,
			CC:             srv.CC,
			ConnectURL:     fmt.Sprintf("steam://connect/%s:%d", srv.Address, srv.Port),
			ConnectCommand: fmt.Sprintf("connect %s:%d", srv.Address, srv.Port),
		},
	}

	for _, target := range sendTargets {
		startPayload.Users = append(startPayload.Users, member{
			Name:    target.user.Name,
			SteamID: target.user.SteamID.String(),
			Hash:    target.user.Avatarhash,
		})
	}

	q.broadcast(Msg{Op: StartGame, Payload: startPayload}, sendTargets...)

	return nil
}

func (q *ServerQueue) Disconnect(client *Client) {
	q.mu.RLock()

	var serverIDs []int //nolint:prealloc
	for server := range q.state {
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

func (q *ServerQueue) Message(message domain.Message, user domain.UserProfile) error {
	message.BodyMD = sanitizeUserMessage(message.BodyMD)
	if len(message.BodyMD) == 0 {
		return ErrBadInput
	}

	id, errID := uuid.NewV7()
	if errID != nil {
		return errors.Join(errID, errMessageID)
	}

	message.ID = id
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

	return nil
}

func (q *ServerQueue) broadcast(payload Msg, targetClients ...*Client) {
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

func (q *ServerQueue) Ping(client *Client) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	client.messageQueue <- Msg{
		Op:      Pong,
		Payload: pongPayload{CreatedOn: time.Now()},
	}
}

func (q *ServerQueue) sendClientChatHistory(client *Client) {
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
