package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/mm"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	// wsMsgTypePugLobbyListStatesRequest  = 1008.
	wsMsgTypePugLobbyListStatesResponse = 1009
	wsMsgTypePugJoinSlotRequest         = 1010
	// wsMsgTypePugJoinSlotResponse        = 1011.
	//
	// // Quickplay
	// wsMsgTypeQPCreateLobbyRequest  = 2000
	// wsMsgTypeQPCreateLobbyResponse = 2001
	// wsMsgTypeQPLeaveLobbyRequest   = 2002
	// wsMsgTypeQPLeaveLobbyResponse  = 2003
	// wsMsgTypeQPJoinLobbyRequest    = 2004
	// wsMsgTypeQPJoinLobbyResponse   = 2005
	// wsMsgTypeQPUserMessageRequest  = 2006
	// wsMsgTypeQPUserMessageResponse = 2007
	//
	// wsMsgTypeErrResponse = 10000.
)

const tokenLen = 6

func newWebSocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

var (
	ErrInvalidLobbyID  = errors.New("Invalid lobby id")
	ErrDuplicateClient = errors.New("Duplicate client")
	ErrUnknownClient   = errors.New("Unknown client")
	ErrLobbyNotEmpty   = errors.New("Lobby is not empty")
	ErrSlotInvalid     = errors.New("Slot invalid")
)

type LobbyType int

const (
	lobbyTypeQuickPlay LobbyType = iota
	lobbyTypePug
)

// LobbyService provides common interface for interacting with multiple lobby types.
type LobbyService interface {
	lobbyType() LobbyType
	join(client *wsClient) error
	joinSlot(client *wsClient, slot string) error
	leave(client *wsClient) error
	sendUserMessage(client *wsClient, msg lobbyUserMessageRequest)
	broadcast(msgType wsMsgType, status bool, payload any)
	clientCount() int
	id() string
}

type wsClient struct {
	sendChan chan wsValue
	logger   *zap.Logger
	socket   *websocket.Conn
	User     model.UserProfile `json:"user"`
	lobbies  []LobbyService
}

func (client *wsClient) currentPugLobby() (*pugLobby, bool) {
	for _, lobby := range client.lobbies {
		if lobby.lobbyType() == lobbyTypePug {
			lobbyVal, ok := lobby.(*pugLobby)
			if !ok {
				return nil, false
			}

			return lobbyVal, true
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
		client.logger.Error("Cannot send client ws payload: channel full")
	}
}

func newWsClient(logger *zap.Logger, socket *websocket.Conn, user model.UserProfile) *wsClient {
	return &wsClient{
		logger:   logger.Named(fmt.Sprintf("ws-%d", user.SteamID.Int64())),
		socket:   socket,
		User:     user,
		sendChan: make(chan wsValue, 5),
	}
}

func (client *wsClient) writer() {
	for {
		payload := <-client.sendChan
		if errSend := client.socket.WriteJSON(payload); errSend != nil {
			client.logger.Error("Failed to send json payload", zap.Error(errSend))

			return
		}
		client.logger.Debug("Wrote client payload", zap.Int("msg_type", int(payload.MsgType)))
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
	LobbyID string `json:"lobby_id"`
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

type wsJoinLobbySlotRequest struct {
	LobbyID string `json:"lobby_id"`
	Slot    string `json:"slot"`
}

type wsRequestHandler func(cm *wsConnectionManager, client *wsClient, payload json.RawMessage) error

type wsConnectionManager struct {
	*sync.RWMutex
	logger      *zap.Logger
	connections wsClients
	lobbies     map[string]LobbyService
	handlers    map[wsMsgType]wsRequestHandler
}

func (cm *wsConnectionManager) pubLobbyList() []*pugLobby {
	lobbies := []*pugLobby{}
	cm.RLock()
	for _, l := range cm.lobbies {
		lobbyVal, ok := l.(*pugLobby)
		if !ok {
			cm.logger.Warn("Failed to cast publobby")

			continue
		}
		lobbies = append(lobbies, lobbyVal)
	}
	cm.RUnlock()

	return lobbies
}

func newWSConnectionManager(ctx context.Context, logger *zap.Logger) *wsConnectionManager {
	connManager := wsConnectionManager{
		RWMutex: &sync.RWMutex{},
		logger:  logger.Named("conman"),
		lobbies: map[string]LobbyService{},
		handlers: map[wsMsgType]wsRequestHandler{
			wsMsgTypePugCreateLobbyRequest: createPugLobby,
			wsMsgTypePugUserMessageRequest: sendPugUserMessage,
			wsMsgTypePugLeaveLobbyRequest:  leavePugLobby,
			wsMsgTypePugJoinLobbyRequest:   joinPugLobby,
			wsMsgTypePugJoinSlotRequest:    joinPugLobbySlot,
		},
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
				cm.RLock()
				cm.connections.broadcast(wsMsgTypePugLobbyListStatesResponse,
					true, wsPugLobbyListStatesResponse{Lobbies: cm.pubLobbyList()})
				cm.RUnlock()
			}
		}
	}(&connManager)

	return &connManager
}

//nolint:ireturn
func (cm *wsConnectionManager) findLobby(lobbyID string) (LobbyService, error) {
	if !lobbyIDValid(lobbyID) {
		return nil, ErrInvalidLobbyID
	}
	cm.RLock()
	defer cm.RUnlock()
	lobby, found := cm.lobbies[lobbyID]
	if !found {
		return nil, ErrInvalidLobbyID
	}

	return lobby, nil
}

type createLobbyOpts struct {
	GameType        mm.GameType   `json:"game_type"`
	GameConfig      mm.GameConfig `json:"game_config"`
	MapName         string        `json:"map_name"`
	Description     string        `json:"description"`
	DiscordRequired bool          `json:"discord_required"`
}

func (cm *wsConnectionManager) createPugLobby(client *wsClient, opts createLobbyOpts) (*pugLobby, error) {
	cm.Lock()
	defer cm.Unlock()
	lobbyID := golib.RandomString(tokenLen)
	_, found := cm.lobbies[lobbyID]
	if found {
		return nil, errors.New("Failed to create unique lobby")
	}
	lobby := newPugLobby(cm.logger, client, lobbyID, opts)
	cm.lobbies[lobbyID] = lobby
	cm.logger.Info("Pug lobby created", zap.String("lobby_id", lobbyID))

	return lobby, nil
}

// func (cm *wsConnectionManager) createQPLobby(client *wsClient) (LobbyService, error) {
//	cm.Lock()
//	defer cm.Unlock()
//	lobbyId := golib.RandomString(tokenLen)
//	_, found := cm.lobbies[lobbyId]
//	if found {
//		return nil, errors.New("Failed to create unique lobby")
//	}
//	lobby := newQPLobby(lobbyId, client)
//	cm.lobbies[lobbyId] = lobby
//	cm.logger.Info("Lobby created")
//	return lobby, nil
// }

func (cm *wsConnectionManager) removeLobby(lobbyID string) error {
	cm.Lock()
	defer cm.Unlock()
	lobby, found := cm.lobbies[lobbyID]
	if !found {
		return ErrInvalidLobbyID
	}
	if lobby.clientCount() > 0 {
		return ErrLobbyNotEmpty
	}
	delete(cm.lobbies, lobbyID)
	cm.logger.Info("Pug lobby deleted", zap.String("lobby_id", lobbyID))

	return nil
}

func (cm *wsConnectionManager) join(client *wsClient) error {
	cm.Lock()
	cm.connections = append(cm.connections, client)
	cm.Unlock()
	cm.logger.Info("New client connection", zap.Int64("sid64", client.User.SteamID.Int64()))
	sendPugLobbyListStates(cm, client, nil)

	return nil
}

func lobbyIDValid(lobbyID string) bool {
	return len(lobbyID) == tokenLen
}

func (cm *wsConnectionManager) leave(client *wsClient) error {
	cm.Lock()
	defer cm.Unlock()
	for _, lobby := range cm.lobbies {
		if errLeave := lobby.leave(client); errLeave != nil {
			cm.logger.Error("Failed to remove client from lobby",
				zap.Int64("sid64", client.User.SteamID.Int64()), zap.Error(errLeave))
		}
	}
	cm.connections = fp.Remove(cm.connections, client)
	cm.logger.Info("Client disconnected", zap.Int64("sid64", client.User.SteamID.Int64()))

	return nil
}

// func (cm *wsConnectionManager) newLobbyId() string {
//	valid := false
//	loops := 0
//	const maxLoops = 100
//	var lobbyId string
//	for !valid && loops <= maxLoops {
//		lobbyId = golib.RandomString(6)
//		lobby, _ := cm.findLobby(lobbyId)
//		valid = lobby == nil
//		loops++
//	}
//	if loops >= maxLoops {
//		panic("Could not generate unique lobby id")
//	}
//	return lobbyId
// }

func (cm *wsConnectionManager) handleMessage(client *wsClient, msgType wsMsgType, payload json.RawMessage) error {
	// TODO split out into map and register handlers instead of mega switch
	handler, handlerFound := cm.handlers[msgType]
	if !handlerFound {
		return errors.New("Unhandled message type")
	}

	return handler(cm, client, payload)
}

func wsConnHandler(w http.ResponseWriter, r *http.Request, cm *wsConnectionManager, user model.UserProfile, logger *zap.Logger) {
	// webSocketUpgrader.CheckOrigin = func(r *http.Request) bool {
	//	origin := r.Header.Get("Origin")
	//	allowed := fp.Contains(config.HTTP.CorsOrigins, origin)
	//	if !allowed {
	//		web.logger.Error("Invalid websocket origin", zap.Error(origin))
	//	}
	//	return allowed
	// }
	log := logger.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	upgrader := newWebSocketUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade websocket", zap.Error(err))

		return
	}
	log.Debug("New connection", zap.String("addr", conn.LocalAddr().String()))
	// New user connection
	// TODO track between connections so they can resume their session upon dc
	client := newWsClient(log, conn, user)

	go client.writer()

	if errJoin := cm.join(client); errJoin != nil {
		log.Error("Failed to join client pool", zap.Error(errJoin))

		return
	}

	defer func() {
		if errLeave := cm.leave(client); errLeave != nil {
			log.Error("Error dropping client", zap.Error(errLeave))
		}
		log.Debug("Client disconnected", zap.Int64("sid64", client.User.SteamID.Int64()))
	}()
	type wsRequest struct {
		MsgType wsMsgType       `json:"msg_type"`
		Status  bool            `json:"status"`
		Payload json.RawMessage `json:"payload"`
	}
	for {
		var basePayload wsRequest
		if errRead := conn.ReadJSON(&basePayload); errRead != nil {
			var wsErr *websocket.CloseError
			ok := errors.Is(errRead, wsErr)
			if !ok {
				log.Error("Unhandled error trying to write ws payload", zap.Error(errRead))
			}
			// else {
			// switch wsErr.Code {
			// case websocket.CloseGoingAway:
			//	// remove client
			// }
			// }
			_ = cm.leave(client)

			return
		}
		if errHandle := cm.handleMessage(client, basePayload.MsgType, basePayload.Payload); errHandle != nil {
			log.Error("Failed to handle ws message", zap.Error(errHandle))
			client.send(basePayload.MsgType+1, false, wsMsgErrorResponse{
				Error: errHandle.Error(),
			})
		}
	}
}
