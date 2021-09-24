package web

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web/ws"
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
	"time"
)

const (
	sendQueueSize = 100
	recvQueueSize = 100
)

type State int32

const (
	Closed State = iota
	Opened
	AwaitingAuthentication
	Authenticated
	Closing
)

// socketState holds the global websocket session state and handlers
type socketState struct {
	*sync.RWMutex
	ws         *melody.Melody
	db         store.Store
	logMsgChan chan LogPayload
	sessions   map[*melody.Session]*clientSession
}

// clientSession represents the state of a client connected via websockets
type clientSession struct {
	IsClient bool
	State    State
	Person   model.Person
	// Is log broadcasting enabled
	BroadcastLog        bool
	LogQueryOpts        model.LogQueryOpts
	LogQueryOptsUpdated bool
	ctx                 context.Context
	eventChan           chan model.ServerEvent
	session             *melody.Session
	sendQ               chan []byte
	recvQ               chan ws.Payload
	log                 *log.Entry
}

func (s *clientSession) send(b []byte) {
	select {
	case s.sendQ <- b:
		break
	default:
		s.log.Errorf("send queue full")
	}
}

func (s *clientSession) writer() {
	for {
		select {
		case p := <-s.sendQ:
			b, errEnc := json.Marshal(p)
			if errEnc != nil {
				log.Errorf("Failed to encode ws payload: %v", errEnc)
				continue
			}
			if err := s.session.Write(b); err != nil {
				s.log.Errorf("Failed to write payload over write: %v", err)
				continue
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// reader sends out incoming log payloads to the client
func (s *clientSession) reader() {
	for {
		select {
		case r := <-s.recvQ:
			s.log.Debugln(r)
		case e := <-s.eventChan:
			if !s.LogQueryOpts.ValidRecordType(e.EventType) {
				continue
			}
			// TODO
			b, err := ws.Encode(ws.LogType, e)
			if err != nil {
				s.log.Errorf("Failed to EncodeWSPayload payload: %v", err)
				continue
			}
			if errE := s.session.Write(b); errE != nil {
				s.log.Errorf("Failed to write to ws: %v", errE)
			}
		case <-s.ctx.Done():
			s.log.Debugf("ws reader() shutdown")
			return
		}
	}
}

func (s *clientSession) setQueryOpts(opts model.LogQueryOpts) {
	s.LogQueryOpts = opts
	s.LogQueryOptsUpdated = true
}

func (s *clientSession) err(errType ws.Type, err error, args ...interface{}) {
	if len(args) == 1 {
		s.log.Errorf(args[0].(string))
	} else if len(args) > 1 {
		s.log.Errorf(args[0].(string), args[1:]...)
	}
	s.send(newWSErr(errType, err))
}

// newClientServiceState allocates and connects all websocket routes and session states
func newClientServiceState(logMsgChan chan LogPayload, db store.Store) *socketState {
	wsWeb := melody.New()
	wss := &socketState{
		RWMutex:    &sync.RWMutex{},
		ws:         wsWeb,
		db:         db,
		sessions:   map[*melody.Session]*clientSession{},
		logMsgChan: logMsgChan,
	}
	wsWeb.HandleMessage(wss.onMessage)
	wsWeb.HandleConnect(wss.onWSConnect)
	wsWeb.HandleDisconnect(wss.onWSDisconnect)
	wsWeb.HandleError(func(session *melody.Session, err error) {
		log.Errorf("WSERR: %v", err)
		// dc?
	})
	return wss
}

func (s *socketState) onWSStart(c *gin.Context) {
	if err := s.ws.HandleRequest(c.Writer, c.Request); err != nil {
		log.Errorf("Error handling s request: %v", err)
	}
}

type SocketAuthReq struct {
	Token      string `json:"token"`
	IsServer   bool   `json:"is_server"`
	ServerName string `json:"server_name"`
}

type WebSocketAuthResp struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type WSErrRes struct {
	Error string `json:"err"`
}

func newWSErr(errType ws.Type, err error) []byte {
	ev := ""
	if err != nil {
		ev = err.Error()
	}
	d, _ := json.Marshal(WSErrRes{Error: ev})
	b, _ := json.Marshal(ws.Payload{
		Type: errType,
		Data: d,
	})
	return b
}

func (s *socketState) authenticateClient(ctx context.Context, req SocketAuthReq, cs *clientSession) error {
	cs.IsClient = true
	sid, err := sid64FromJWTToken(req.Token)
	if err != nil {
		return consts.ErrAuthentication
	}
	var p model.Person
	if errP := s.db.GetPersonBySteamID(ctx, sid, &p); errP != nil || p.PermissionLevel < model.PModerator {
		return consts.ErrAuthentication
	}

	cs.Person = p

	b, errEnc := ws.Encode(ws.AuthOKType, WebSocketAuthResp{
		Status:  true,
		Message: "Successfully authenticated",
	})
	if errEnc != nil {
		cs.log.Errorf("Failed to encode auth response payload: %v", errEnc)
		return consts.ErrAuthentication
	}
	cs.send(b)
	cs.log.Debugf("WS user authhenticated successfully")

	return nil
}

func (s *socketState) onMessage(session *melody.Session, msg []byte) {
	s.Lock()
	defer s.Unlock()
	sockSession, found := s.sessions[session]
	if !found {
		log.Errorf("Unknown s client sent message")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()

	var w ws.Payload
	if err := json.Unmarshal(msg, &w); err != nil {
		sockSession.err(ws.ErrType, consts.ErrMalformedRequest, "Failed to unmarshal s payload")
		return
	}
	s.onAuthenticatedPayload(ctx, &w, sockSession)
}

func (s *socketState) onAuthenticatedPayload(_ context.Context, w *ws.Payload, c *clientSession) {
	switch w.Type {
	case ws.LogType:
		var l LogPayload
		if err := json.Unmarshal(w.Data, &l); err != nil {
			c.err(ws.ErrType, consts.ErrMalformedRequest, "Failed to unmarshal logpayload data")
			return
		}
		s.logMsgChan <- l
	case ws.LogQueryOpts:
		var opts model.LogQueryOpts
		if err := json.Unmarshal(w.Data, &opts); err != nil {
			c.err(ws.ErrType, consts.ErrMalformedRequest, "Failed to unmarshal query data")
			return
		}
		c.setQueryOpts(opts)
		c.log.Debugf("Updated query opts: %v", opts)
		go func() {
			results, err := s.db.FindLogEvents(c.ctx, opts)
			if err != nil {
				c.log.Errorf("Error sending pre-cache to client")
				return
			}
			for _, r := range results {
				b, e := ws.Encode(ws.LogQueryResults, r)
				if e != nil {
					c.log.Errorf("Failed to encode payload: %v", e)
					return
				}
				c.send(b)
			}
		}()
	default:
		c.log.Debugf("Unhandled payload: %v", w)
	}
}

// onWSConnect sets up the websocket client in the session list and registers it to receive all log events
// by default.
func (s *socketState) onWSConnect(session *melody.Session) {
	client := &clientSession{
		State:     Closed,
		ctx:       context.Background(),
		eventChan: make(chan model.ServerEvent),
		session:   session,
		sendQ:     make(chan []byte, sendQueueSize),
		recvQ:     make(chan ws.Payload),
		log:       log.WithFields(log.Fields{"addr": session.Request.RemoteAddr}),
	}
	go client.reader()
	go client.writer()
	s.Lock()
	s.sessions[session] = client
	s.Unlock()
	client.log.Infof("WS client connect")

}

// onWSDisconnect will remove the client from the active session list and unregister itself
// from the event broadcasts
func (s *socketState) onWSDisconnect(session *melody.Session) {
	s.Lock()
	defer s.Unlock()
	c, found := s.sessions[session]
	if !found {
		log.Errorf("Unregistered s client")
		return
	}
	c.State = Closing
	delete(s.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
	if err := event.UnregisterConsumer(c.eventChan); err != nil {
		log.Errorf("Failed to unregister event consumer")
	}
	// TODO flush remaining queues
	c.State = Closed
}
