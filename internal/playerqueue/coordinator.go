package playerqueue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

// TODO Track a users desired minimum size for them to be counted towards play.
// Show users queue size for both at their min levels and without.

type Coordinator struct {
	chatLogHistorySize int
	minQueueSize       int
	lobbies            []*Lobby
	clients            []domain.QueueClient
	chatLogs           []domain.ChatLog
	mu                 *sync.RWMutex
	currentState       func() ([]Lobby, error)
}

func New(chatLogHistorySize int, minQueueSize int, chatlogs []domain.ChatLog, currentStateFunc func() ([]Lobby, error)) *Coordinator {
	return &Coordinator{
		minQueueSize:       minQueueSize,
		clients:            []domain.QueueClient{},
		chatLogs:           chatlogs,
		lobbies:            []*Lobby{},
		mu:                 &sync.RWMutex{},
		chatLogHistorySize: chatLogHistorySize,
		currentState:       currentStateFunc,
	}
}

func (q *Coordinator) Start(ctx context.Context) {
	cleanupTicker := time.NewTicker(time.Second * 30)
	refreshState := time.NewTicker(time.Second * 2)

	for {
		select {
		case <-cleanupTicker.C:
			q.removeZombies()
		case <-refreshState.C:
			state, errUpdate := q.currentState()
			if errUpdate != nil {
				slog.Error("Failed to update state", log.ErrAttr(errUpdate))

				continue
			}

			q.UpdateState(state)

			if err := q.checkQueueCompat(); err != nil {
				slog.Error("Failed to check queue compatibility", log.ErrAttr(err))
			}
		case <-ctx.Done():
			q.broadcast(domain.Response{Op: domain.Bye, Payload: byePayload{Message: "Server shutting down... run!!!"}})

			return
		}
	}
}

func (q *Coordinator) removeZombies() {
	q.mu.Lock()
	defer q.mu.Unlock()

	var valid []domain.QueueClient
	for _, client := range q.clients {
		if client.IsTimedOut() {
			q.removeFromQueues(client)
			client.Close()
			slog.Debug("Removing zombie client", slog.String("client", client.ID()))
		} else {
			valid = append(valid, client)
		}
	}

	q.clients = valid
}

func (q *Coordinator) updateClientStates(fullUpdate bool) {
	update := clientStatePayload{}

	q.mu.RLock()
	defer q.mu.RUnlock()

	//goland:noinspection GoPreferNilSlice
	updateMap := []queueState{}
	for _, value := range q.lobbies {
		updateMap = append(updateMap, queueState{
			ServerID: value.ServerID,
			Members:  value.Members,
		})
	}

	update.UpdateServers = true
	update.Servers = updateMap

	if fullUpdate {
		//goland:noinspection GoPreferNilSlice
		players := []member{}
		for _, client := range q.clients {
			sid := client.SteamID()
			players = append(players, member{
				Name:    client.Name(),
				SteamID: sid.String(),
				Hash:    client.Avatarhash(),
			})
		}
		update.UpdateUsers = true
		update.Users = players
	}

	q.broadcast(domain.Response{Op: domain.StateUpdate, Payload: update})
}

func (q *Coordinator) Leave(client domain.QueueClient, servers []int) error {
	changed := false

	q.mu.Lock()
	for _, serverID := range servers {
		for _, srv := range q.lobbies {
			if srv.ServerID != serverID {
				continue
			}

			//goland:noinspection GoPreferNilSlice
			valid := []ClientQueueState{}
			for _, mem := range srv.Members {
				if mem.SteamID != client.SteamID() {
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

func (q *Coordinator) Join(client domain.QueueClient, servers []int) error {
	changed := false
	q.mu.Lock()

	for _, serverID := range servers {
		for _, srv := range q.lobbies {
			if srv.ServerID != serverID {
				continue
			}

			found := false
			for _, mem := range srv.Members {
				if mem.SteamID == client.SteamID() {
					found = true
				}
			}

			if !found {
				srv.Members = append(srv.Members, ClientQueueState{SteamID: client.SteamID()})
				changed = true
			}

			break
		}
	}

	q.mu.Unlock()

	if changed {
		q.updateClientStates(false)

		return q.checkQueueCompat()
	}

	return nil
}

// Connect adds the user to the swarm. If a user exists with the same steamid exists, it will be replaced with
// the new connection.
func (q *Coordinator) Connect(ctx context.Context, steamID steamid.SteamID, name string, avatarHash string, conn *websocket.Conn) domain.QueueClient {
	q.mu.Lock()
	defer q.mu.Unlock()

	client := newClient(steamID, name, avatarHash, conn)

	for i := range q.clients {
		if q.clients[i].SteamID() == client.SteamID() {
			q.clients[i].Close()

			break
		}
	}

	q.clients = append(q.clients, client)

	go client.Start(ctx)
	if client.HasMessageAccess() {
		go q.sendClientChatHistory(client)
	}
	go q.updateClientStates(true)

	return client
}

func (q *Coordinator) UpdateState(lobbies []Lobby) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var valid []*Lobby

	for _, lobby := range lobbies {
		found := false

		for _, existingLobby := range q.lobbies {
			if lobby.ServerID == existingLobby.ServerID {
				existingLobby.PlayerCount = lobby.PlayerCount
				existingLobby.MaxPlayers = lobby.MaxPlayers
				valid = append(valid, existingLobby)

				found = true

				break
			}
		}

		if !found {
			// Create new entries for missing keys.
			valid = append(valid, &Lobby{
				ServerID:    lobby.ServerID,
				PlayerCount: lobby.PlayerCount,
				MaxPlayers:  lobby.MaxPlayers,
				Members:     make([]ClientQueueState, 0),
			})
		}
	}

	q.lobbies = valid
}

func (q *Coordinator) checkQueueCompat() error {
	var serverID int

	q.mu.Lock()
	for _, lobby := range q.lobbies {
		if len(lobby.Members) < q.minQueueSize {
			continue
		}

		if lobby.MaxPlayers-lobby.PlayerCount-len(lobby.Members) < 0 {
			continue
		}

		serverID = lobby.ServerID

		break
	}
	q.mu.Unlock()

	if serverID > 0 {
		return q.initiateGame(serverID)
	}

	return nil
}

func (q *Coordinator) initiateGame(serverID int) error {
	q.mu.RLock()
	var queuedClients []domain.QueueClient

	var currentLobby *Lobby

	for _, lobby := range q.lobbies {
		if lobby.ServerID != serverID {
			continue
		}

		currentLobby = lobby

		break
	}

	if currentLobby == nil {
		return ErrFindLobby
	}

	// Find the queued users via their matching steamid
	for _, client := range q.clients {
		for _, c := range currentLobby.Members {
			if client.SteamID() == c.SteamID {
				queuedClients = append(queuedClients, client)

				break
			}
		}
	}

	ipAddr, errIP := currentLobby.IP()
	if errIP != nil {
		return errIP
	}

	startPayload := gameStartPayload{
		Server: server{
			Name:           currentLobby.Title,
			ShortName:      currentLobby.ShortName,
			CC:             currentLobby.CC,
			ConnectURL:     fmt.Sprintf("steam://connect/%s:%d", ipAddr.String(), currentLobby.Port),
			ConnectCommand: fmt.Sprintf("connect %s:%d", currentLobby.Hostname, currentLobby.Port),
		},
	}

	for _, target := range queuedClients {
		sid := target.SteamID()
		startPayload.Users = append(startPayload.Users, member{
			Name:    target.Name(),
			SteamID: sid.String(),
			Hash:    target.Avatarhash(),
		})
	}

	q.broadcast(domain.Response{Op: domain.StartGame, Payload: startPayload}, queuedClients...)
	q.mu.RUnlock()

	q.mu.Lock()
	for _, client := range queuedClients {
		q.removeFromQueues(client)
	}
	q.mu.Unlock()

	q.updateClientStates(false)

	return nil
}

func (q *Coordinator) FindMessages(steamID steamid.SteamID, limit int) []domain.ChatLog {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var messages []domain.ChatLog
	for i := len(q.chatLogs) - 1; i >= 0; i-- {
		if q.chatLogs[i].SteamID == steamID {
			messages = append(messages, q.chatLogs[i])
			if len(messages) == limit {
				return messages
			}
		}
	}

	return messages
}

func (q *Coordinator) PurgeMessages(deletedIDs ...int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove the purged messages from the local cache.
	var valid []domain.ChatLog
	for _, existing := range q.chatLogs {
		if !slices.Contains(deletedIDs, existing.MessageID) {
			valid = append(valid, existing)
		}
	}
	q.chatLogs = valid

	q.broadcast(domain.Response{Op: domain.Purge, Payload: purgePayload{MessageIDs: deletedIDs}})
}

// Disconnect removes a client from the coordinator entirely, leaving all its queues, closing the underlying client and broadcasting
// the full state change to all clients.
func (q *Coordinator) Disconnect(client domain.QueueClient) {
	q.mu.RLock()

	var serverIDs []int //nolint:prealloc
	for serverQueue := range q.lobbies {
		serverIDs = append(serverIDs, serverQueue)
	}

	q.mu.RUnlock()

	if err := q.Leave(client, serverIDs); err != nil {
		slog.Warn("Error leaving server queue", log.ErrAttr(err))
	}

	q.mu.Lock()
	var valid []domain.QueueClient //nolint:prealloc
	for _, existing := range q.clients {
		if existing.SteamID() == client.SteamID() {
			continue
		}

		valid = append(valid, existing)
	}

	q.clients = valid
	q.mu.Unlock()

	q.updateClientStates(true)

	client.Close()
}

// Message adds a new chat log entry and broadcasts the entry to all elidgable clients.
func (q *Coordinator) Message(message domain.ChatLog) {
	q.mu.Lock()
	q.chatLogs = append(q.chatLogs, message)
	if len(q.chatLogs) > q.chatLogHistorySize {
		q.chatLogs = q.chatLogs[1:]
	}
	q.mu.Unlock()

	q.mu.RLock()
	q.broadcast(domain.Response{
		Op:      domain.Message,
		Payload: message,
	})
	q.mu.RUnlock()
}

// broadcast sends a domain.Response payload to multiple clients. If no clients are specified, all
// clients will receive the payload.
func (q *Coordinator) broadcast(payload domain.Response, targetClients ...domain.QueueClient) {
	if len(targetClients) == 0 {
		targetClients = q.clients
	}

	for _, client := range targetClients {
		slog.Debug("Sending message to client", slog.Int("op", int(payload.Op)),
			slog.String("client", client.ID()))
		// Make sure we skip physically sending messages to clients without at least read access to the chat messages.
		if payload.Op == domain.Message && !client.HasMessageAccess() {
			continue
		}
		go client.Send(payload)
	}
}

func (q *Coordinator) removeFromQueues(client domain.QueueClient) {
	for _, srv := range q.lobbies {
		var valid []ClientQueueState

		for _, mem := range srv.Members {
			if mem.SteamID != client.SteamID() {
				valid = append(valid, mem)
			}
		}

		srv.Members = valid
	}
}

// sendClientChatHistory sends the last N messages of the chatLogs history to the client provided.
func (q *Coordinator) sendClientChatHistory(client domain.QueueClient) {
	q.mu.RLock()
	msgs := make([]domain.Response, len(q.chatLogs))
	for i, cl := range q.chatLogs {
		msgs[i] = domain.Response{
			Op:      domain.Message,
			Payload: cl,
		}
	}
	q.mu.RUnlock()

	for _, msg := range msgs {
		client.Send(msg)
	}
}

// UpdateChatStatus updates a client with their new ChatStatus.
func (q *Coordinator) UpdateChatStatus(steamID steamid.SteamID, status domain.ChatStatus, reason string, previous domain.ChatStatus) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, client := range q.clients {
		if client.SteamID() != steamID {
			continue
		}

		client.Send(domain.Response{
			Op:      domain.ChatStatusChange,
			Payload: domain.ChatStatusChangePayload{Status: status, Reason: reason},
		})

		if previous == domain.Noaccess {
			go q.sendClientChatHistory(client)
		}

		break
	}
}
