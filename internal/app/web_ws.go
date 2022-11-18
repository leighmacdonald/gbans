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
	"time"
)

type wsMsgType int

const (
	wsMsgTypePugCreateLobbyRequest  = 1000
	wsMsgTypePugCreateLobbyResponse = 1001
	wsMsgTypePugLeaveLobbyRequest   = 1002
	wsMsgTypePugLeaveLobbyResponse  = 1003
	wsMsgTypePugJoinLobbyRequest    = 1004
	wsMsgTypePugJoinLobbyResponse   = 1005
	wsMsgTypePugUserMessageRequest  = 1006
	wsMsgTypePugUserMessageResponse = 1007

	wsMsgTypePugLobbyListStatesRequest  = 1008
	wsMsgTypePugLobbyListStatesResponse = 1009

	// Quickplay
	wsMsgTypeQPCreateLobbyRequest  = 2000
	wsMsgTypeQPCreateLobbyResponse = 2001
	wsMsgTypeQPLeaveLobbyRequest   = 2002
	wsMsgTypeQPLeaveLobbyResponse  = 2003
	wsMsgTypeQPJoinLobbyRequest    = 2004
	wsMsgTypeQPJoinLobbyResponse   = 2005
	wsMsgTypeQPUserMessageRequest  = 2006
	wsMsgTypeQPUserMessageResponse = 2007

	wsMsgTypeErrResponse = 10000
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
	ErrInvalidHandler  = errors.New("Invalid handler")
)

type LobbyType int

const (
	lobbyTypeQuickPlay LobbyType = iota
	lobbyTypePug
)

// LobbyService provides common interface for interacting with multiple lobby types
type LobbyService interface {
	lobbyType() LobbyType
	join(client *wsClient) error
	leave(client *wsClient) error
	sendUserMessage(client *wsClient, msg lobbyUserMessageRequest)
	broadcast(msgType wsMsgType, status bool, payload any)
	clientCount() int
	id() string
}

type wsClient struct {
	sendChan chan wsValue
	socket   *websocket.Conn
	User     model.UserProfile `json:"user"`
	ctx      context.Context
	lobbies  []LobbyService
}

func (client *wsClient) currentPugLobby() (*pugLobby, bool) {
	for _, lobby := range client.lobbies {
		if lobby.lobbyType() == lobbyTypePug {
			return lobby.(*pugLobby), true
		}
	}
	return nil, false
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
func (client *wsClient) send(msgType wsMsgType, status bool, payload any) {
	select {
	case client.sendChan <- wsValue{
		MsgType: msgType,
		Status:  status,
		Payload: payload,
	}:
	default:
		log.Warnf("Cannot send client ws payload: channel full")
	}

}
func newWsClient(ctx context.Context, socket *websocket.Conn, user model.UserProfile) *wsClient {
	return &wsClient{ctx: ctx, socket: socket, User: user, sendChan: make(chan wsValue, 5)}
}

func (client *wsClient) writer() {
	for {
		select {
		case payload := <-client.sendChan:
			if errSend := client.socket.WriteJSON(payload); errSend != nil {
				log.WithError(errSend).Errorf("Failed to send json payload")
				return
			}
			log.WithFields(log.Fields{"msg_type": payload.MsgType}).Debugf("Wrote client payload")
		case <-client.ctx.Done():
			return
		}
	}
}

type wsClients []*wsClient

func (clients wsClients) broadcast(msgType wsMsgType, status bool, payload any) {
	for _, client := range clients {
		client.send(msgType, status, payload)
	}
}

type wsValue struct {
	MsgType wsMsgType `json:"msg_type"`
	Status  bool      `json:"status"`
	Payload any       `json:"payload"`
}

type wsJoinLobbyRequest struct {
	LobbyId string `json:"lobby_id"`
}

type wsJoinLobbyResponse struct {
	Lobby *pugLobby `json:"lobby"`
}

type wsPugLobbyListStatesResponse struct {
	Lobbies []*pugLobby `json:"lobbies"`
}

type pugUserMessageResponse struct {
	User      model.UserProfile `json:"user"`
	Message   string            `json:"message"`
	CreatedAt time.Time         `json:"created_at"`
}
type lobbyUserMessageRequest struct {
	Message string `json:"message"`
}

type wsMsgErrorResponse struct {
	Error string `json:"error"`
}

type wsBroadcastFn func(msgType wsMsgType, status bool, payload any)

type wsRequestHandler func(cm *wsConnectionManager, client *wsClient, payload json.RawMessage) error

type wsConnectionManager struct {
	*sync.RWMutex
	connections wsClients
	lobbies     map[string]LobbyService
	handlers    map[wsMsgType]wsRequestHandler
}

func (cm *wsConnectionManager) pubLobbyList() []*pugLobby {
	lobbies := []*pugLobby{}
	cm.RLock()
	for _, l := range cm.lobbies {
		lobbies = append(lobbies, l.(*pugLobby))
	}
	cm.RUnlock()
	return lobbies
}

func newWSConnectionManager(ctx context.Context) *wsConnectionManager {
	wsHandlers := map[wsMsgType]wsRequestHandler{
		wsMsgTypePugCreateLobbyRequest: createPugLobby,
		wsMsgTypePugUserMessageRequest: sendPugUserMessage,
		wsMsgTypePugLeaveLobbyRequest:  leavePugLobby,
		wsMsgTypePugJoinLobbyRequest:   joinPugLobby,
	}
	connManager := wsConnectionManager{
		RWMutex:     &sync.RWMutex{},
		lobbies:     map[string]LobbyService{},
		handlers:    wsHandlers,
		connections: nil,
	}
	go func(cm *wsConnectionManager) {
		// Send lobby state updates periodically to all clients
		timer := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				cm.connections.broadcast(wsMsgTypePugLobbyListStatesResponse,
					true, wsPugLobbyListStatesResponse{Lobbies: cm.pubLobbyList()})
			}
		}
	}(&connManager)
	return &connManager
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

type createLobbyOpts struct {
	GameType        string `json:"game_type"`
	GameConfig      string `json:"game_config"`
	MapName         string `json:"map_name"`
	Description     string `json:"description"`
	DiscordRequired bool   `json:"discord_required"`
}

func (cm *wsConnectionManager) createPugLobby(client *wsClient, opts createLobbyOpts) (*pugLobby, error) {
	cm.Lock()
	defer cm.Unlock()
	lobbyId := golib.RandomString(tokenLen)
	_, found := cm.lobbies[lobbyId]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby, errLobby := newPugLobby(client, lobbyId, opts)
	if errLobby != nil {
		return nil, errLobby
	}
	cm.lobbies[lobbyId] = lobby
	log.WithFields(log.Fields{"lobby_id": lobbyId}).
		Info("Pug lobby created")
	return lobby, nil
}

//func (cm *wsConnectionManager) createQPLobby(client *wsClient) (LobbyService, error) {
//	cm.Lock()
//	defer cm.Unlock()
//	lobbyId := golib.RandomString(tokenLen)
//	_, found := cm.lobbies[lobbyId]
//	if found {
//		return nil, errors.New("Failed to create unique lobby")
//	}
//	lobby := newQPLobby(lobbyId, client)
//	cm.lobbies[lobbyId] = lobby
//	log.WithFields(log.Fields{"lobby_id": lobbyId}).
//		Info("Lobby created")
//	return lobby, nil
//}

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
	cm.connections = append(cm.connections, client)
	cm.Unlock()
	log.WithFields(log.Fields{"steam_id": client.User.SteamID}).
		Info("New client connection")
	sendPugLobbyListStates(cm, client, nil)
	return nil
}

func lobbyIdValid(lobbyId string) bool {
	return len(lobbyId) == tokenLen
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
	loops := 0
	const maxLoops = 100
	var lobbyId string
	for !valid && loops <= maxLoops {
		lobbyId = golib.RandomString(6)
		lobby, _ := cm.findLobby(lobbyId)
		valid = lobby == nil
		loops++
	}
	if loops >= maxLoops {
		panic("Could not generate unique lobby id")
	}
	return lobbyId
}

func (cm *wsConnectionManager) handleMessage(client *wsClient, msgType wsMsgType, payload json.RawMessage) error {
	// TODO split out into map and register handlers instead of mega switch
	handler, handlerFound := cm.handlers[msgType]
	if !handlerFound {
		return errors.New("Unhandled message type")
	}
	return handler(cm, client, payload)
}

func (web *web) wsConnHandler(w http.ResponseWriter, r *http.Request, user model.UserProfile) {
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

	if errJoin := web.cm.join(client); errJoin != nil {
		log.WithError(errJoin).Error("Failed to join client pool")
	}

	defer func() {
		if errLeave := web.cm.leave(client); errLeave != nil {
			log.WithError(errLeave).Errorf("Error dropping client")
		}
		log.WithFields(log.Fields{"steam_id": client.User.SteamID.String()}).
			Debugf("Client disconnected")
	}()
	type wsRequest struct {
		MsgType wsMsgType       `json:"msg_type"`
		Status  bool            `json:"status"`
		Payload json.RawMessage `json:"payload"`
	}
	for {
		var basePayload wsRequest
		if errRead := conn.ReadJSON(&basePayload); errRead != nil {
			wsErr, ok := errRead.(*websocket.CloseError)
			if !ok {
				log.WithError(errRead).Error("Unhandled error trying to write ws payload")
			} else {
				switch wsErr.Code {
				case websocket.CloseGoingAway:
					// remove client
				}
			}
			_ = web.cm.leave(client)
			return
		}
		if errHandle := web.cm.handleMessage(client, basePayload.MsgType, basePayload.Payload); errHandle != nil {
			log.WithError(errHandle).Error("Failed to handle ws message")
			client.send(basePayload.MsgType+1, false, wsMsgErrorResponse{
				Error: errHandle.Error(),
			})
		}
	}
}
