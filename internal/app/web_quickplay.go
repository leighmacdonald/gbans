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
	"golang.org/x/exp/slices"
	"net/http"
	"sync"
)

type qpMsgType int

const (
	qpMsgTypeJoinLobbyRequest = iota
	qpMsgTypeLeaveLobbyRequest
	qpMsgTypeJoinLobbySuccess
	qpMsgTypeSendMsgRequest
	qpMsgTypeErrResponse
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

type qpClient struct {
	send   chan qpBaseResponse
	Leader bool `json:"leader"`
	socket *websocket.Conn
	User   model.UserProfile `json:"user"`
	lobby  *qpLobby
	ctx    context.Context
}

func newQpClient(ctx context.Context, socket *websocket.Conn, user model.UserProfile) *qpClient {
	return &qpClient{ctx: ctx, socket: socket, User: user, send: make(chan qpBaseResponse)}
}

func (client *qpClient) writer() {
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

type qpClients []*qpClient

func (clients qpClients) broadcast(payload qpBaseResponse) {
	for _, client := range clients {
		client.send <- payload
	}
}

type qpLobby struct {
	*sync.RWMutex
	LobbyId  string          `json:"lobby_id"`
	Clients  qpClients       `json:"clients"`
	Messages []qpUserMessage `json:"messages"`
}

func newLobby(lobbyId string) *qpLobby {
	return &qpLobby{
		RWMutex:  &sync.RWMutex{},
		LobbyId:  lobbyId,
		Clients:  qpClients{},
		Messages: []qpUserMessage{},
	}
}

func (lobby *qpLobby) join(client *qpClient) error {
	lobby.Lock()
	defer lobby.Unlock()
	if slices.Contains(lobby.Clients, client) {
		return ErrDuplicateClient
	}
	client.lobby = lobby
	lobby.Clients = append(lobby.Clients, client)
	log.WithFields(log.Fields{
		"clients": len(lobby.Clients),
		"leader":  len(lobby.Clients) == 1,
		"lobby":   lobby.LobbyId,
	}).Infof("User joined lobby")
	if len(lobby.Clients) == 1 {
		return lobby.promote(client)
	}
	return nil
}

func (lobby *qpLobby) leave(client *qpClient) error {
	lobby.Lock()
	defer lobby.Unlock()
	if !slices.Contains(lobby.Clients, client) {
		return ErrUnknownClient
	}
	if len(lobby.Clients) == 1 {
		return ErrEmptyLobby
	}
	lobby.Clients = fp.Remove(lobby.Clients, client)
	client.lobby = nil
	if client.Leader {
		client.Leader = false
		return lobby.promote(lobby.Clients[0])
	}
	return nil
}

func (lobby *qpLobby) promote(client *qpClient) error {
	for _, lobbyClient := range lobby.Clients {
		client.Leader = lobbyClient == client
	}
	return nil
}

func (lobby *qpLobby) SendUserMessage(msg qpUserMessage) {
	lobby.Lock()
	defer lobby.Unlock()
	lobby.Messages = append(lobby.Messages, msg)
}

func (lobby *qpLobby) broadcast(response qpBaseResponse) error {
	for _, client := range lobby.Clients {
		client.send <- response
	}
	return nil
}

func (cm *qpConnectionManager) findLobby(lobbyId string) (*qpLobby, error) {
	cm.RLock()
	defer cm.RUnlock()
	lobby, found := cm.lobbies[lobbyId]
	if !found {
		return nil, ErrInvalidLobbyId
	}
	return lobby, nil
}

func (cm *qpConnectionManager) createLobby() (*qpLobby, error) {
	cm.Lock()
	defer cm.Unlock()
	lobbyId := golib.RandomString(tokenLen)
	_, found := cm.lobbies[lobbyId]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby := newLobby(lobbyId)
	cm.lobbies[lobbyId] = lobby
	log.WithFields(log.Fields{"lobby_id": lobbyId}).
		Info("Lobby created")
	return lobby, nil
}

func (cm *qpConnectionManager) removeLobby(lobbyId string) error {
	cm.Lock()
	defer cm.Unlock()
	lobby, found := cm.lobbies[lobbyId]
	if !found {
		return ErrInvalidLobbyId
	}
	if len(lobby.Clients) > 0 {
		return ErrLobbyNotEmpty
	}
	delete(cm.lobbies, lobbyId)
	log.WithFields(log.Fields{"lobby_id": lobbyId}).
		Info("Lobby deleted")
	return nil
}

func (cm *qpConnectionManager) join(client *qpClient) error {
	cm.Lock()
	defer cm.Unlock()
	cm.connections = append(cm.connections, client)
	log.WithFields(log.Fields{"steam_id": client.User.SteamID}).
		Info("New client connection")
	return nil
}

func (cm *qpConnectionManager) leave(client *qpClient) error {
	cm.Lock()
	defer cm.Unlock()
	if client.lobby != nil {
		if errLeave := client.lobby.leave(client); errLeave != nil {
			log.Errorf("Failed to cleanup user from lobby")
		}
	}
	cm.connections = fp.Remove(cm.connections, client)
	log.WithFields(log.Fields{"steam_id": client.User.SteamID}).
		Infof("Client disconnected")
	return nil
}

func qpHandleWSMessage(cm *qpConnectionManager, client *qpClient, payload qpBasePayload) error {
	switch payload.MsgType {
	case qpMsgTypeLeaveLobbyRequest:
		if errLeave := client.lobby.leave(client); errLeave != nil {
			return errLeave
		}
		if client.lobby == nil {
			cm.createAndJoinLobby(client)
		}
		return nil
	case qpMsgTypeJoinLobbyRequest:
		var req qpJoinLobbyRequestPayload
		if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
		}
		if len(req.LobbyId) != tokenLen {
			return ErrInvalidLobbyId
		}
		lobby, lobbyErr := cm.findLobby(req.LobbyId)
		if lobbyErr != nil {
			return lobbyErr
		}
		if errJoin := lobby.join(client); errJoin != nil {
			return errJoin
		}
		if errResp := lobby.broadcast(qpBaseResponse{
			MsgType: qpMsgTypeJoinLobbySuccess,
			Payload: qpMsgJoinedLobbySuccess{
				Lobby: lobby,
			},
		}); errResp != nil {
			log.WithError(errResp).
				Error("Failed to send join lobby success response")
		}
		return nil
	case qpMsgTypeSendMsgRequest:
		var userMessage qpUserMessage
		if errUnmarshal := json.Unmarshal(payload.Payload, &userMessage); errUnmarshal != nil {
			log.WithError(errUnmarshal).Error("Failed to unmarshal msg request")
			return errUnmarshal
		}
		client.lobby.SendUserMessage(userMessage)
		return nil
	default:
		return errors.New("Unknown message type")
	}

}

func sendJoinLobbySuccess(client *qpClient, lobby *qpLobby) {
	client.send <- qpBaseResponse{
		qpMsgTypeJoinLobbySuccess,
		qpMsgJoinedLobbySuccess{
			Lobby: lobby,
		},
	}
}

func (cm *qpConnectionManager) createAndJoinLobby(client *qpClient) {
	// Create and join a lobby for the user
	lobby, lobbyErr := cm.createLobby()
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

func qpWSHandler(w http.ResponseWriter, r *http.Request, cm *qpConnectionManager, user model.UserProfile) {
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
	client := newQpClient(r.Context(), conn, user)
	go client.writer()
	if errJoin := cm.join(client); errJoin != nil {
		log.WithError(errJoin).Error("Failed to join client pool")
	}
	cm.createAndJoinLobby(client)
	defer func() {
		if errLeave := cm.leave(client); errLeave != nil {
			log.WithError(errLeave).Errorf("Error dropping client")
		}
		log.WithFields(log.Fields{"steam_id": client.User.SteamID.String()}).
			Debugf("Client disconnected")
	}()
	for {
		var basePayload qpBasePayload
		if errRead := conn.ReadJSON(&basePayload); errRead != nil {
			log.WithError(errRead).Error("Failed to read json payload")
			return
		}
		if errHandle := qpHandleWSMessage(cm, client, basePayload); errHandle != nil {
			log.WithError(errHandle).Error("Failed to handle message")
			client.send <- qpBaseResponse{
				qpMsgTypeErrResponse,
				qpMsgErrorResponse{
					Error: errHandle.Error(),
				},
			}
		}
	}
}
