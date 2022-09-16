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
	qpMsgTypeJoin = iota
	qpMsgTypeLeave
	qpMsgTypeJoinLobby
	qpMsgTypeSendMsg
)

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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
	LobbyId  string          `json:"lobby_id"`
	Clients  qpClients       `json:"clients"`
	Messages []qpUserMessage `json:"messages"`
}

func (clients qpLobby) SendUserMessage(msg qpUserMessage) {
	clients.Messages = append(clients.Messages, msg)
	clients.Clients.broadcast(qpBaseResponse{
		MsgType: qpMsgTypeSendMsg,
		Payload: msg,
	})
}

type qpBaseResponse struct {
	MsgType qpMsgType `json:"msg_type"`
	Payload any       `json:"payload"`
}

type qpBasePayload struct {
	MsgType qpMsgType       `json:"msg_type"`
	Payload json.RawMessage `json:"payload"`
}

type qpConnectionManager struct {
	*sync.RWMutex
	connections qpClients
	lobbies     map[string]*qpLobby
}

type qpJoinRequest struct {
	LobbyId string `json:"lobby_id"`
}

type qpUserMessage struct {
	SteamId   string `json:"steam_id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type qpJoinLobbyRequest struct {
	Lobby *qpLobby `json:"lobby"`
}

func (cm *qpConnectionManager) createLobby(client *qpClient) (*qpLobby, error) {
	lobbyId := golib.RandomString(6)
	_, found := cm.lobbies[lobbyId]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby := qpLobby{
		LobbyId:  lobbyId,
		Clients:  qpClients{client},
		Messages: []qpUserMessage{},
	}
	cm.lobbies[lobbyId] = &lobby
	return &lobby, nil
}

func (cm *qpConnectionManager) join(client *qpClient) error {
	cm.connections = append(cm.connections, client)
	lobby, lobbyErr := cm.createLobby(client)
	if lobbyErr != nil {
		return lobbyErr
	}
	client.Lobby = lobby
	body, errBody := json.Marshal(qpJoinLobbyRequest{Lobby: lobby})
	if errBody != nil {
		return errBody
	}
	return client.send(qpBasePayload{
		MsgType: qpMsgTypeJoinLobby,
		Payload: body,
	})
}

func (cm *qpConnectionManager) leave(client *qpClient) error {
	cm.connections = fp.Remove(cm.connections, client)
	return nil
}

func qpHandleWSMessage(cm *qpConnectionManager, client *qpClient, payload qpBasePayload) error {
	switch payload.MsgType {
	case qpMsgTypeJoin:
		{
			var req qpJoinRequest
			if errUnmarshal := json.Unmarshal(payload.Payload, &req); errUnmarshal != nil {
				log.WithError(errUnmarshal).Error("Failed to unmarshal join request")
			}
			if errJoin := cm.join(client); errJoin != nil {
				return errJoin
			}
			return nil
		}
	case qpMsgTypeSendMsg:
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
	client := newQpClient(conn, user)
	if errJoin := cm.join(client); errJoin != nil {
		log.WithError(errJoin).Error("Failed to join client pool")
	}
	for {
		var basePayload qpBasePayload
		if errRead := conn.ReadJSON(&basePayload); errRead != nil {
			log.WithError(errRead).Error("Failed to read json payload")
			continue
		}
		if errHandle := qpHandleWSMessage(cm, client, basePayload); errHandle != nil {
			log.WithError(errHandle).Error("Failed to handle message")
		}
	}
}
