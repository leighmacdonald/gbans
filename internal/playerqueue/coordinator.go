package playerqueue

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// TODO Track a users desired minimum size for them to be counted towards play.
// Show users queue size for both at their min levels and without.

type Coordinator struct {
	chatLogHistorySize int
	minQueueSize       int
	lobbies            []*Lobby
	clients            []Client
	chatLogs           []ChatLog
	mu                 *sync.RWMutex
	validLobbies       func() ([]Lobby, error)
}

func New(chatLogHistorySize int, minQueueSize int, chatlogs []ChatLog, currentStateFunc func() ([]Lobby, error)) *Coordinator {
	return &Coordinator{
		minQueueSize:       minQueueSize,
		clients:            []Client{},
		chatLogs:           chatlogs,
		lobbies:            []*Lobby{},
		mu:                 &sync.RWMutex{},
		chatLogHistorySize: chatLogHistorySize,
		validLobbies:       currentStateFunc,
	}
}

func (q *Coordinator) updateState() {
	lobbies, errUpdate := q.validLobbies()
	if errUpdate != nil {
		slog.Error("Failed to update state", slog.String("error", errUpdate.Error()))

		return
	}

	q.replaceLobbies(lobbies)

	if err := q.checkQueueCompat(); err != nil {
		slog.Error("Failed to check queue compatibility", slog.String("error", err.Error()))
	}
}

func (q *Coordinator) updateClientStates(fullUpdate bool) {
	update := ClientStatePayload{}

	q.mu.RLock()
	defer q.mu.RUnlock()

	//goland:noinspection GoPreferNilSlice
	updateMap := []LobbyState{}
	for _, value := range q.lobbies {
		updateMap = append(updateMap, LobbyState{
			ServerID: value.ServerID,
			Members:  value.Members,
		})
	}

	update.UpdateServers = true
	update.Lobbies = updateMap

	if fullUpdate {
		//goland:noinspection GoPreferNilSlice
		players := []Member{}
		for _, client := range q.clients {
			sid := client.SteamID()
			players = append(players, Member{
				Name:    client.Name(),
				SteamID: sid.String(),
				Hash:    client.Avatarhash(),
			})
		}
		update.UpdateUsers = true
		update.Users = players
	}

	go q.broadcast(Response{Op: StateUpdate, Payload: update})
}

func (q *Coordinator) Leave(client Client, servers []int) error {
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
		go q.updateClientStates(false)
	}

	return nil
}

func (q *Coordinator) Join(client Client, servers []int) error {
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
func (q *Coordinator) Connect(ctx context.Context, steamID steamid.SteamID, name string, avatarHash string, conn *websocket.Conn) Client { //nolint:ireturn
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

func (q *Coordinator) replaceLobbies(lobbies []Lobby) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var valid []*Lobby

	for _, lobby := range lobbies {
		found := false

		for _, existingLobby := range q.lobbies {
			if lobby.ServerID == existingLobby.ServerID {
				lobby.Members = existingLobby.Members
				valid = append(valid, &lobby)

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
	var queuedClients []Client

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

	startPayload := GameStartPayload{
		Server: LobbyServer{
			Name:           currentLobby.Title,
			ShortName:      currentLobby.ShortName,
			CC:             currentLobby.CC,
			ConnectURL:     fmt.Sprintf("steam://connect/%s:%d", ipAddr.String(), currentLobby.Port),
			ConnectCommand: fmt.Sprintf("connect %s:%d", currentLobby.Hostname, currentLobby.Port),
		},
	}

	for _, target := range queuedClients {
		sid := target.SteamID()
		startPayload.Users = append(startPayload.Users, Member{
			Name:    target.Name(),
			SteamID: sid.String(),
			Hash:    target.Avatarhash(),
		})
	}

	q.broadcast(Response{Op: StartGame, Payload: startPayload}, queuedClients...)
	q.mu.RUnlock()

	q.mu.Lock()
	for _, client := range queuedClients {
		q.removeFromQueues(client)
	}
	q.mu.Unlock()

	q.updateClientStates(false)

	return nil
}

func (q *Coordinator) FindMessages(steamID steamid.SteamID, limit int) []ChatLog {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var messages []ChatLog
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
	var valid []ChatLog
	for _, existing := range q.chatLogs {
		if !slices.Contains(deletedIDs, existing.MessageID) {
			valid = append(valid, existing)
		}
	}
	q.chatLogs = valid

	q.broadcast(Response{Op: Purge, Payload: PurgePayload{MessageIDs: deletedIDs}})
}

// Disconnect removes a client from the coordinator entirely, leaving all its queues, closing the underlying client and broadcasting
// the full state change to all clients.
func (q *Coordinator) Disconnect(client Client) {
	q.mu.RLock()

	var serverIDs []int //nolint:prealloc
	for serverQueue := range q.lobbies {
		serverIDs = append(serverIDs, serverQueue)
	}

	q.mu.RUnlock()

	if err := q.Leave(client, serverIDs); err != nil {
		slog.Warn("Error leaving server queue", slog.String("error", err.Error()))
	}

	q.mu.Lock()
	var valid []Client //nolint:prealloc
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

// Message adds a new chat log entry and broadcasts the entry to all eligible clients.
func (q *Coordinator) Message(message ChatLog) {
	q.mu.Lock()
	q.chatLogs = append(q.chatLogs, message)
	if len(q.chatLogs) > q.chatLogHistorySize {
		q.chatLogs = q.chatLogs[1:]
	}
	q.mu.Unlock()

	q.mu.RLock()
	q.broadcast(Response{
		Op: Message,
		Payload: MessagePayload{
			Messages: []ChatLog{message},
		},
	})
	q.mu.RUnlock()
}

// broadcast sends a domain.Response payload to multiple clients. If no clients are specified, all
// clients will receive the payload.
func (q *Coordinator) broadcast(payload Response, targetClients ...Client) {
	if len(targetClients) == 0 {
		targetClients = q.clients
	}

	for _, client := range targetClients {
		slog.Debug("Sending message to client", slog.Int("op", int(payload.Op)),
			slog.String("client", client.ID()))
		// Make sure we skip physically sending messages to clients without at least read access to the chat messages.
		if payload.Op == Message && !client.HasMessageAccess() {
			continue
		}
		go client.Send(payload)
	}
}

func (q *Coordinator) removeFromQueues(client Client) {
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
func (q *Coordinator) sendClientChatHistory(client Client) {
	payload := MessagePayload{
		Messages: []ChatLog{},
	}

	q.mu.RLock()
	payload.Messages = append(payload.Messages, q.chatLogs...)
	q.mu.RUnlock()

	go client.Send(Response{
		Op:      Message,
		Payload: payload,
	})
}

// UpdateChatStatus updates a client with their new ChatStatus.
func (q *Coordinator) UpdateChatStatus(steamID steamid.SteamID, status ChatStatus, reason string, previous ChatStatus) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, client := range q.clients {
		if client.SteamID() != steamID {
			continue
		}

		go client.Send(Response{
			Op:      ChatStatusChange,
			Payload: ChatStatusChangePayload{Status: status, Reason: reason},
		})

		if previous == Noaccess {
			go q.sendClientChatHistory(client)
		}

		break
	}
}
