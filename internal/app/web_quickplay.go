package app

import (
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

type qpMsgType int

const (
	qpMsgTypeJoinLobbyRequest = iota
	qpMsgTypeLeaveLobbyRequest
	qpMsgTypeJoinLobbySuccess
	qpMsgTypeSendMsgRequest
)

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var (
	ErrInvalidLobbyId  = errors.New("Invalid lobby id")
	ErrDuplicateClient = errors.New("Duplicate client")
)

type qpClient struct {
	Leader     bool `json:"leader"`
	socket     *websocket.Conn
	sync.Mutex `json:"-"`
	User       model.UserProfile `json:"user"`
	Lobby      *qpLobby          `json:"-"`
}

func (client *qpClient) send(value any) error {
	client.Lock()
	defer client.Unlock()
	return client.socket.WriteJSON(value)
}

func newQpClient(socket *websocket.Conn, user model.UserProfile) *qpClient {
	return &qpClient{socket: socket, User: user, Mutex: sync.Mutex{}}
}

type qpClients []*qpClient

func (clients qpClients) broadcast(value any) {
	for _, client := range clients {
		if errSend := client.send(value); errSend != nil {
			log.Errorf("failed to send ws payload: %s", errSend)
		}
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
	if fp.Contains(lobby.Clients, client) {
		return ErrDuplicateClient
	}
	lobby.Clients = append(lobby.Clients, client)
	if len(lobby.Clients) == 1 {
		return lobby.promote(client)
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
	lobby.Messages = append(lobby.Messages, msg)
	// TODO do at ws layer
	lobby.Clients.broadcast(qpBaseResponse{
		MsgType: qpMsgTypeSendMsgRequest,
		Payload: msg,
	})
}

func (cm *qpConnectionManager) findLobby(lobbyId string) (*qpLobby, error) {
	lobby, found := cm.lobbies[lobbyId]
	if !found {
		return nil, ErrInvalidLobbyId
	}
	return lobby, nil
}

func (cm *qpConnectionManager) createLobby(client *qpClient) (*qpLobby, error) {
	lobbyId := golib.RandomString(6)
	_, found := cm.lobbies[lobbyId]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby := newLobby(lobbyId)
	cm.lobbies[lobbyId] = lobby
	return lobby, nil
}

func (cm *qpConnectionManager) join(client *qpClient) error {
	cm.connections = append(cm.connections, client)
	return nil
}

func (cm *qpConnectionManager) leave(client *qpClient) error {
	cm.connections = fp.Remove(cm.connections, client)
	return nil
}

func qpHandleWSMessage(cm *qpConnectionManager, client *qpClient, payload qpBasePayload) error {
	switch payload.MsgType {
	case qpMsgTypeJoinLobbyRequest:
		{
			var req qpJoinLobbyRequestPayload
			if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
				log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
			}
			lobby, lobbyErr := cm.findLobby(req.LobbyId)
			if lobbyErr != nil {
				return lobbyErr
			}
			lobby.join(client)
			if errJoin := cm.join(client); errJoin != nil {
				return errJoin
			}
			return nil
		}

	case qpMsgTypeSendMsgRequest:
		{
			var userMessage qpUserMessage
			if errUnmarshal := json.Unmarshal(payload.Payload, &userMessage); errUnmarshal != nil {
				log.WithError(errUnmarshal).Error("Failed to unmarshal msg request")
				return errUnmarshal
			}
			client.Lobby.SendUserMessage(userMessage)
			return nil
		}
	}

	return errors.New("Unknown message type")
}

func sendJoinLobbySuccess(client *qpClient, lobby *qpLobby) error {
	return client.send(qpBaseResponse{
		qpMsgTypeJoinLobbySuccess,
		qpMsgJoinedLobbySuccess{Lobby: lobby},
	})
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
	client := newQpClient(conn, user)
	if errJoin := cm.join(client); errJoin != nil {
		log.WithError(errJoin).Error("Failed to join client pool")
	}
	// Create and join a lobby for the user
	lobby, lobbyErr := cm.createLobby(client)
	if lobbyErr != nil {
		log.WithError(lobbyErr).Error("Failed to create lobby")
		return
	}
	if errJoin := lobby.join(client); errJoin != nil {
		log.WithError(lobbyErr).Error("Failed to join lobby")
		return
	}

	if errSend := sendJoinLobbySuccess(client, lobby); errSend != nil {
		log.WithError(errSend).Error("Failed to send join lobby payload")
		return
	}

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
			if errError := client.send(qpMsgErrorResponse{Error: errHandle.Error()}); errError != nil {
				log.WithError(errError).Error("Failed to send error response to client")
			}
		}
	}
}
