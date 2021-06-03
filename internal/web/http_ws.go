package web

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
	"time"
)

type payloadType int

const (
	OKType       payloadType = iota
	ErrType                  = 1
	AuthType                 = 2
	AuthOKType               = 3
	LogType                  = 4
	LogQueryOpts             = 5
)

// WebSocketPayload represents the basic structure of all websocket requests. Decoding is a 2 stage
// process as we must first know the payload_type before we can decode the Data value into the appropriate
// struct.
type WebSocketPayload struct {
	PayloadType payloadType     `json:"payload_type"`
	Data        json.RawMessage `json:"data"`
}

type WebSocketLogPayload struct {
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
}

// webSocketState holds the global websocket session state and handlers
type webSocketState struct {
	*sync.RWMutex
	ws       *melody.Melody
	sessions map[*melody.Session]*webSocketSession
}

// webSocketSession represents the state of a client connected via websockets
type webSocketSession struct {
	authenticated       bool
	Person              *model.Person
	BroadcastLog        bool
	open                bool
	LogQueryOpts        webSocketLogQueryOpts
	LogQueryOptsUpdated bool
	ctx                 context.Context
	eventChan           chan model.LogEvent
	session             *melody.Session
}

type webSocketLogQueryOpts struct {
	LogTypes []logparse.MsgType `json:"log_types"`
	Limit    int                `json:"limit"`
	SourceID steamid.SID64      `json:"source_id"`
	TargetID steamid.SID64      `json:"target_id"`
	Servers  []int              `json:"servers"`
}

func (lqo *webSocketLogQueryOpts) okRecordType(t logparse.MsgType) bool {
	if len(lqo.LogTypes) == 0 {
		// No filters == Any
		return true
	}
	for _, mt := range lqo.LogTypes {
		if mt == t {
			return true
		}
	}
	return false
}

// EncodeWSPayload will return an encoded payload suitable for transmission over the wire
func EncodeWSPayload(t payloadType, p interface{}) ([]byte, error) {
	b, e1 := json.Marshal(p)
	if e1 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload base payload")
	}
	f, e2 := json.Marshal(WebSocketPayload{
		PayloadType: t,
		Data:        b,
	})
	if e2 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload sub payload")
	}
	return f, nil
}

// reader sends out incoming log payloads to the client
func (c *webSocketSession) reader() {
	for {
		select {
		case e := <-c.eventChan:
			if !c.LogQueryOpts.okRecordType(e.Type) {
				continue
			}
			b, err := EncodeWSPayload(LogType, e)
			if err != nil {
				log.Errorf("Failed to EncodeWSPayload payload: %v", err)
				continue
			}
			if errE := c.session.Write(b); errE != nil {
				log.Errorf("Failed to write to ws: %v", errE)
			}
		case <-c.ctx.Done():
			log.Debugf("ws reader() shutdown")
			return
		}
	}
}

func (c *webSocketSession) setQueryOpts(opts webSocketLogQueryOpts) {
	c.LogQueryOpts = opts
	c.LogQueryOptsUpdated = true
}

// newWebSocketState allocates and connects all websocket routes and session states
func newWebSocketState() *webSocketState {
	ws := melody.New()
	wss := &webSocketState{
		RWMutex:  &sync.RWMutex{},
		ws:       ws,
		sessions: map[*melody.Session]*webSocketSession{},
	}
	ws.HandleMessage(wss.onMessage)
	ws.HandleConnect(wss.onWSConnect)
	ws.HandleDisconnect(wss.onWSDisconnect)
	ws.HandleError(func(session *melody.Session, err error) {
		log.Errorf("WSERR: %v", err)
		// dc?
	})
	return wss
}

func (ws *webSocketState) onWSStart(c *gin.Context) {
	if err := ws.ws.HandleRequest(c.Writer, c.Request); err != nil {
		log.Errorf("Error handling ws request: %v", err)
	}
}

type WebSocketAuthReq struct {
	Token      string `json:"token"`
	IsServer   bool   `json:"is_server"`
	ServerName string `json:"server_name"`
}

type WebSocketAuthResp struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type wsErrRes struct {
	Error string `json:"error"`
}

func newWSErr(err error) []byte {
	d, _ := json.Marshal(wsErrRes{Error: err.Error()})
	b, _ := json.Marshal(WebSocketPayload{
		PayloadType: ErrType,
		Data:        d,
	})
	return b
}

func authenticateServer(ctx context.Context, req WebSocketAuthReq, session *webSocketSession) error {
	if req.Token == "" || req.ServerName == "" {
		return consts.ErrAuthhentication
	}
	s, e := store.GetServerByName(ctx, req.ServerName)
	if e != nil {
		return consts.ErrAuthhentication
	}
	if s.Password == "" {
		log.Errorf("Server has empty password!!!")
		return consts.ErrAuthhentication
	}
	if req.Token != s.Password {
		log.Errorf("Invalid password used for server auth")
		return consts.ErrAuthhentication
	}
	b, errEnc := EncodeWSPayload(AuthOKType, WebSocketAuthResp{
		Status:  true,
		Message: "Successfully authenticated",
	})
	if errEnc != nil {
		log.Errorf("Failed to encode auth response payload: %v", errEnc)
		return consts.ErrAuthhentication
	}
	if err := session.session.Write(b); err != nil {
		log.Errorf("Failed to write client success response: %v", err)
	}
	log.WithFields(log.Fields{"server_name": s.ServerName}).Debugf("WS server authhenticated successfully")
	return nil
}

func authenticateClient(ctx context.Context, req WebSocketAuthReq, session *webSocketSession) error {
	sid, err := sid64FromJWTToken(req.Token)
	if err != nil {
		return consts.ErrAuthhentication
	}
	p, errP := store.GetPersonBySteamID(ctx, sid)
	if errP != nil || p.PermissionLevel < model.PModerator {
		return consts.ErrAuthhentication
	}
	session.Person = p

	b, errEnc := EncodeWSPayload(AuthOKType, WebSocketAuthResp{
		Status:  true,
		Message: "Successfully authenticated",
	})
	if errEnc != nil {
		log.Errorf("Failed to encode auth response payload: %v", errEnc)
		return consts.ErrAuthhentication
	}
	if err := session.session.Write(b); err != nil {
		log.Errorf("Failed to write client success response: %v", err)
	}
	log.WithFields(log.Fields{"steam_id": p.SteamID}).
		Debugf("WS user authhenticated successfully")
	return nil
}

// onMessage handles incoming websocket payloads
// We always return authentication errors until the client is fully authed. This is to prevent
// any leaking of information to an attacker that can be further leveraged to aide in further
// attacks by this or other vectors.
func (ws *webSocketState) onMessage(session *melody.Session, msg []byte) {
	ws.Lock()
	defer ws.Unlock()
	wsClientSession, found := ws.sessions[session]
	if !found {
		log.Errorf("Unknown ws client sent message")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var w WebSocketPayload
	if err := json.Unmarshal(msg, &w); err != nil || !wsClientSession.authenticated && w.PayloadType != AuthType {
		log.Errorf("Failed to unmarshal ws payload")
		_ = session.Write(newWSErr(consts.ErrAuthhentication))
		return
	}
	if !wsClientSession.authenticated {
		var req WebSocketAuthReq
		if err := json.Unmarshal(w.Data, &req); err != nil {
			log.Errorf("Failed to unmarshal auth data")
			_ = session.Write(newWSErr(consts.ErrAuthhentication))
			return
		}
		var e error
		if req.IsServer {
			e = authenticateServer(ctx, req, wsClientSession)
		} else {
			e = authenticateClient(ctx, req, wsClientSession)
		}
		if e != nil {
			if err := session.Write(newWSErr(consts.ErrAuthhentication)); err != nil {
				log.Errorf("Failed to write client error: %v", err)
			}
			return
		}
		wsClientSession.authenticated = true
	} else {
		switch w.PayloadType {
		case LogQueryOpts:
			var tok webSocketLogQueryOpts
			if err := json.Unmarshal(w.Data, &tok); err != nil {
				log.Errorf("Failed to unmarshal query data")
				_ = session.Write(newWSErr(consts.ErrAuthhentication))
				return
			}
			wsClientSession.setQueryOpts(tok)
		}
	}
}

// onWSConnect sets up the websocket client in the session list and registers it to receive all log events
// by default.
func (ws *webSocketState) onWSConnect(session *melody.Session) {
	client := &webSocketSession{
		ctx:           context.Background(),
		eventChan:     make(chan model.LogEvent),
		open:          true,
		session:       session,
		authenticated: false,
	}
	go client.reader()
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client connect")
	ws.Lock()
	ws.sessions[session] = client
	ws.Unlock()
}

// onWSDisconnect will remove the client from the active session list and unregister itself
// from the event broadcasts
func (ws *webSocketState) onWSDisconnect(session *melody.Session) {
	ws.Lock()
	defer ws.Unlock()
	c, found := ws.sessions[session]
	if !found {
		log.Errorf("Unregistered ws client")
		return
	}
	c.open = false
	delete(ws.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
	if err := event.UnregisterConsumer(c.eventChan); err != nil {
		log.Errorf("Failed to unregister event consumer")
	}
}
