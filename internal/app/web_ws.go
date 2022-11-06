package app

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

type wsMsgType int

const (
	wsMsgTypeJoinLobbyRequest = iota
	wsMsgTypeLeaveLobbyRequest
	wsMsgTypeJoinLobbySuccess
	wsMsgTypeSendMsgRequest
	wsMsgTypeErrResponse
	wsMsgTypeCreateLobbyRequest
)

const tokenLen = 6

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	ErrInvalidLobbyId  = errors.New("Invalid lobby id")
	ErrDuplicateClient = errors.New("Duplicate client")
	ErrUnknownClient   = errors.New("Unknown client")
	ErrEmptyLobby      = errors.New("Trying to leave empty lobby")
	ErrLobbyNotEmpty   = errors.New("Lobby is not empty")
)

type LobbyType int

const (
	lobbyTypeQuickPlay LobbyType = iota
	lobbyTypePug
)

type LobbyService interface {
	join(client *wsClient) error
	leave(client *wsClient) error
	sendUserMessage(msg wsUserMessage)
	broadcast(response wsBaseResponse) error
	clientCount() int
	id() string
}

type wsClient struct {
	send    chan wsBaseResponse
	socket  *websocket.Conn
	User    model.UserProfile `json:"user"`
	ctx     context.Context
	lobbies []LobbyService
}

func (client *wsClient) removeLobby(lobby LobbyService) {
	var nl []LobbyService
	for _, l := range client.lobbies {
		if l != lobby {
			nl = append(nl, l)
		}
	}
	client.lobbies = nl
}

func newWsClient(ctx context.Context, socket *websocket.Conn, user model.UserProfile) *wsClient {
	return &wsClient{ctx: ctx, socket: socket, User: user, send: make(chan wsBaseResponse)}
}

func (client *wsClient) writer() {
	for {
		select {
		case payload := <-client.send:
			if errSend := client.socket.WriteJSON(payload); errSend != nil {
				log.WithError(errSend).Errorf("Failed to send json payload")
			}
			log.WithFields(log.Fields{"msg_type": payload.MsgType}).Debugf("Wrote client payload")
		case <-client.ctx.Done():
			return
		}
	}
}

type wsClients []*wsClient

func (clients wsClients) broadcast(payload wsBaseResponse) {
	for _, client := range clients {
		client.send <- payload
	}
}

type wsBaseResponse struct {
	MsgType wsMsgType `json:"msg_type"`
	Payload any       `json:"payload"`
}

type wsBasePayload struct {
	MsgType    wsMsgType       `json:"msg_type"`
	Payload    json.RawMessage `json:"payload"`
	MsgSubType int             `json:"msg_sub_type"`
}

type wsCreateLobbyRequest struct {
	LobbyType LobbyType `json:"lobby_type"`
}

type wsJoinLobbyRequest struct {
	LobbyId string `json:"lobby_id"`
}

type wsLeaveLobbyRequest struct {
	LobbyId string `json:"lobby_id"`
}

type wsUserMessage struct {
	LobbyId   string `json:"lobby_id"`
	SteamId   string `json:"steam_id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type wsMsgJoinedLobbySuccess struct {
	LobbyId string `json:"lobby_id"`
}

type wsMsgErrorResponse struct {
	Error string `json:"error"`
}

type wsConnectionManager struct {
	*sync.RWMutex
	connections wsClients
	lobbies     map[string]LobbyService
}

func (cm *wsConnectionManager) findLobby(lobbyId string) (LobbyService, error) {
	if !lobbyIdValid(lobbyId) {
		return nil, ErrInvalidLobbyId
	}
	cm.RLock()
	defer cm.RUnlock()
	lobby, found := cm.lobbies[lobbyId]
	if !found {
		return nil, ErrInvalidLobbyId
	}
	return lobby, nil
}

func (cm *wsConnectionManager) createQPLobby(client *wsClient) (LobbyService, error) {
	cm.Lock()
	defer cm.Unlock()
	lobbyId := golib.RandomString(tokenLen)
	_, found := cm.lobbies[lobbyId]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby := newQPLobby(lobbyId, client)
	cm.lobbies[lobbyId] = lobby
	log.WithFields(log.Fields{"lobby_id": lobbyId}).
		Info("Lobby created")
	return lobby, nil
}

func (cm *wsConnectionManager) removeLobby(lobbyId string) error {
	cm.Lock()
	defer cm.Unlock()
	lobby, found := cm.lobbies[lobbyId]
	if !found {
		return ErrInvalidLobbyId
	}
	if lobby.clientCount() > 0 {
		return ErrLobbyNotEmpty
	}
	delete(cm.lobbies, lobbyId)
	log.WithFields(log.Fields{"lobby_id": lobbyId}).
		Info("Lobby deleted")
	return nil
}

func (cm *wsConnectionManager) join(client *wsClient) error {
	cm.Lock()
	defer cm.Unlock()
	cm.connections = append(cm.connections, client)
	log.WithFields(log.Fields{"steam_id": client.User.SteamID}).
		Info("New client connection")
	return nil
}

func lobbyIdValid(lobbyId string) bool {
	if len(lobbyId) != tokenLen {
		return false
	}
	return true
}

func (cm *wsConnectionManager) leave(client *wsClient) error {
	cm.Lock()
	defer cm.Unlock()
	for _, lobby := range cm.lobbies {
		_ = lobby.leave(client)
	}
	cm.connections = fp.Remove(cm.connections, client)
	log.WithFields(log.Fields{"steam_id": client.User.SteamID}).
		Infof("Client disconnected")
	return nil
}

func (cm *wsConnectionManager) newLobbyId() string {
	cm.RLock()
	defer cm.RUnlock()
	valid := false
	var lobbyId string
	for !valid {
		lobbyId = golib.RandomString(6)
		lobby, _ := cm.findLobby(lobbyId)
		valid = lobby == nil
	}
	return lobbyId
}

func (cm *wsConnectionManager) handleWSMessage(client *wsClient, payload wsBasePayload) error {
	switch payload.MsgType {
	case wsMsgTypeLeaveLobbyRequest:
		var req wsLeaveLobbyRequest
		if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
		}
		if !lobbyIdValid(req.LobbyId) {
			return ErrInvalidLobbyId
		}
		lobby, lobbyErr := cm.findLobby(req.LobbyId)
		if lobbyErr != nil {
			return lobbyErr
		}
		if errJoin := lobby.leave(client); errJoin != nil {
			return errJoin
		}
		if lobby.clientCount() == 0 {
			cm.removeLobby(lobby.id())
		}
		return nil
	case wsMsgTypeCreateLobbyRequest:
		var req wsCreateLobbyRequest
		if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
		}
		var lobby LobbyService
		lobbyId := cm.newLobbyId()
		switch req.LobbyType {
		case lobbyTypePug:
			lobby = newPugLobby(client, lobbyId)
		case lobbyTypeQuickPlay:
			lobby = newQPLobby(lobbyId, client)
		default:
			return errors.New("Unsupported lobby type")
		}
		cm.Lock()
		cm.lobbies[lobby.id()] = lobby
		cm.Unlock()

		if errJoin := lobby.join(client); errJoin != nil {
			return errJoin
		}

		if errResp := lobby.broadcast(wsBaseResponse{
			MsgType: wsMsgTypeJoinLobbySuccess,
			Payload: wsMsgJoinedLobbySuccess{
				LobbyId: lobby.id(),
			},
		}); errResp != nil {
			log.WithError(errResp).
				Error("Failed to send join lobby success response")
		}
		return nil
	case wsMsgTypeJoinLobbyRequest:
		var req wsJoinLobbyRequest
		if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
		}
		lobby, lobbyErr := cm.findLobby(req.LobbyId)
		if lobbyErr != nil {
			return lobbyErr
		}
		if errJoin := lobby.join(client); errJoin != nil {
			return errJoin
		}
		if errResp := lobby.broadcast(wsBaseResponse{
			MsgType: wsMsgTypeJoinLobbySuccess,
			Payload: wsMsgJoinedLobbySuccess{
				LobbyId: lobby.id(),
			},
		}); errResp != nil {
			log.WithError(errResp).
				Error("Failed to send join lobby success response")
		}
		return nil
	case wsMsgTypeSendMsgRequest:
		var userMessage wsUserMessage
		if errUnmarshal := json.Unmarshal(payload.Payload, &userMessage); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal msg request")
			return errUnmarshal
		}
		lobby, lobbyErr := cm.findLobby(userMessage.LobbyId)
		if lobbyErr != nil {
			return lobbyErr
		}
		lobby.sendUserMessage(userMessage)
		return nil
	default:
		return errors.New("Unknown message type")
	}
}

func (cm *wsConnectionManager) createAndJoinLobby(client *wsClient) {
	// Create and join a lobby for the user
	lobby, lobbyErr := cm.createQPLobby(client)
	if lobbyErr != nil {
		log.WithError(lobbyErr).Error("Failed to create lobby")
		return
	}
	if errJoin := lobby.join(client); errJoin != nil {
		log.WithError(lobbyErr).Error("Failed to join lobby")
		return
	}
	sendJoinLobbySuccess(client, lobby)
}

func wsConnHandler(w http.ResponseWriter, r *http.Request, cm *wsConnectionManager, user model.UserProfile) {
	//webSocketUpgrader.CheckOrigin = func(r *http.Request) bool {
	//	origin := r.Header.Get("Origin")
	//	allowed := fp.Contains(config.HTTP.CorsOrigins, origin)
	//	if !allowed {
	//		log.Errorf("Invalid websocket origin: %s", origin)
	//	}
	//	return allowed
	//}
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade websocket: %v", err)
		return
	}
	log.WithFields(log.Fields{"conn": conn.LocalAddr().String()}).
		Debugf("New connection")
	// New user connection
	// TODO track between connections so they can resume their session upon dc
	client := newWsClient(r.Context(), conn, user)

	go client.writer()

	if errJoin := cm.join(client); errJoin != nil {
		log.WithError(errJoin).Error("Failed to join client pool")
	}

	defer func() {
		if errLeave := cm.leave(client); errLeave != nil {
			log.WithError(errLeave).Errorf("Error dropping client")
		}
		log.WithFields(log.Fields{"steam_id": client.User.SteamID.String()}).
			Debugf("Client disconnected")
	}()

	for {
		var basePayload wsBasePayload
		if errRead := conn.ReadJSON(&basePayload); errRead != nil {
			log.WithError(errRead).Error("Failed to read json payload")
			return
		}
		if errHandle := cm.handleWSMessage(client, basePayload); errHandle != nil {
			log.WithError(errHandle).Error("Failed to handle message")
			client.send <- wsBaseResponse{
				wsMsgTypeErrResponse,
				wsMsgErrorResponse{
					Error: errHandle.Error(),
				},
			}
		}
	}
}
