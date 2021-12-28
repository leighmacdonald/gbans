package web

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
	"time"
)

type connState int32

const (
	closed connState = iota
	closing
)

// socketService holds the global websocket session state and handlers
type socketService struct {
	*sync.RWMutex
	ws         *melody.Melody
	db         store.Store
	sessions   map[*melody.Session]*clientSession
	handlers   Handlers
	logMsgChan chan LogPayload
}

// clientSession represents the state of a client connected via websockets
type clientSession struct {
	IsClient bool
	State    connState
	Person   model.Person
	// Is log broadcasting enabled
	BroadcastLog        bool
	LogQueryOpts        model.LogQueryOpts
	LogQueryOptsUpdated bool
	ctx                 context.Context
	eventChan           chan model.ServerEvent
	session             *melody.Session
	sendQ               chan Payload
	recvQ               chan Payload
	log                 *log.Entry
	lastPing            time.Time
}

func (s *clientSession) send(b Payload) {
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
			b, err := Encode(LogType, e)
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

const (
	sendQueueSize = 100
	recvQueueSize = 100
)

func (s *clientSession) setQueryOpts(opts model.LogQueryOpts) {
	s.LogQueryOpts = opts
	s.LogQueryOptsUpdated = true
}

func (s *clientSession) err(errType Type, err error, args ...interface{}) {
	if len(args) == 1 {
		s.log.Errorf(args[0].(string))
	} else if len(args) > 1 {
		s.log.Errorf(args[0].(string), args[1:]...)
	}
	s.send(newWSErr(errType, err))
}

// NewService allocates and connects all websocket routes and session states
func NewService(handlers Handlers, logMsgChan chan LogPayload) *socketService {
	ws := melody.New()
	service := &socketService{
		RWMutex:    &sync.RWMutex{},
		ws:         ws,
		sessions:   map[*melody.Session]*clientSession{},
		handlers:   handlers,
		logMsgChan: logMsgChan,
	}
	ws.HandleMessage(service.onMessage)
	ws.HandleConnect(service.onConnect)
	ws.HandleDisconnect(service.onDisconnect)
	ws.HandleError(func(session *melody.Session, err error) {
		log.Errorf("WSERR: %v", err)
		// dc?
	})
	return service
}

// Start is the websocket api handler entry point
func (s *socketService) Start() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.ws.HandleRequest(c.Writer, c.Request); err != nil {
			log.Errorf("Error handling s request: %v", err)
		}
	}
}

//func (s *socketService) authenticateClient(ctx context.Context, req SocketAuthReq, cs *clientSession) error {
//	cs.IsClient = true
//	sid, err := sid64FromJWTToken(req.Token)
//	if err != nil {
//		return consts.ErrAuthentication
//	}
//	var p model.Person
//	if errP := s.db.GetPersonBySteamID(ctx, sid, &p); errP != nil || p.PermissionLevel < model.PModerator {
//		return consts.ErrAuthentication
//	}
//
//	cs.Person = p
//
//	b, errEnc := ws.Encode(ws.AuthOKType, WebSocketAuthResp{
//		Status:  true,
//		Message: "Successfully authenticated",
//	})
//	if errEnc != nil {
//		cs.log.Errorf("Failed to encode auth response payload: %v", errEnc)
//		return consts.ErrAuthentication
//	}
//	cs.send(b)
//	cs.log.Debugf("WS user authhenticated successfully")
//
//	return nil
//}

func (s *socketService) onMessage(session *melody.Session, msg []byte) {
	s.Lock()
	defer s.Unlock()
	sockSession, found := s.sessions[session]
	if !found {
		log.Errorf("Unknown ws client sent message")
		return
	}
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	//defer cancel()

	var w Payload
	if err := json.Unmarshal(msg, &w); err != nil {
		sockSession.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal s payload")
		return
	}
	s.onAuthenticatedPayload(&w, sockSession)
}

func (s *socketService) onAuthenticatedPayload(w *Payload, c *clientSession) {
	switch w.Type {
	case Sup:
		// TODO cleanup timedout clients
		c.lastPing = config.Now()
		var l Ping
		if err := json.Unmarshal(w.Data, &l); err != nil {
			c.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal logpayload data")
			return
		}
		c.log.WithField("nonce", l.Nonce).Debugf("Got ping")
	case SrvLogRaw:
		var l LogPayload
		if err := json.Unmarshal(w.Data, &l); err != nil {
			c.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal logpayload data")
			return
		}
		log.Debugf("Got log payload (%s): %v", l.ServerName, l.Message)
		s.logMsgChan <- l
	case LogQueryOpts:
		var opts model.LogQueryOpts
		if err := json.Unmarshal(w.Data, &opts); err != nil {
			c.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal query data")
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
				b, e := json.Marshal(r)
				if e != nil {
					c.log.Errorf("Failed to encode payload: %v", e)
					return
				}
				c.send(Payload{
					Type: LogQueryResults,
					Data: b,
				})
			}
		}()
	default:
		c.log.Debugf("Unhandled payload: %v", w)
	}
}

// onConnect sets up the websocket client in the session list and registers it to receive all log events
// by default.
func (s *socketService) onConnect(session *melody.Session) {
	client := &clientSession{
		State:     closed,
		ctx:       context.Background(),
		eventChan: make(chan model.ServerEvent),
		session:   session,
		sendQ:     make(chan Payload, sendQueueSize),
		recvQ:     make(chan Payload, recvQueueSize),
		log:       log.WithFields(log.Fields{"addr": session.Request.RemoteAddr}),
	}
	go client.reader()
	go client.writer()
	s.Lock()
	s.sessions[session] = client
	s.Unlock()
	client.log.Infof("WS client connect")

}

// onDisconnect will remove the client from the active session list and unregister itself
// from the event broadcasts
func (s *socketService) onDisconnect(session *melody.Session) {
	s.Lock()
	defer s.Unlock()
	c, found := s.sessions[session]
	if !found {
		log.Errorf("Unregistered s client")
		return
	}
	c.State = closing
	delete(s.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
	if err := event.UnregisterConsumer(c.eventChan); err != nil {
		log.Errorf("Failed to unregister event consumer")
	}
	// TODO flush remaining queues
	c.State = closed
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

func newWSErr(errType Type, err error) Payload {
	ev := ""
	if err != nil {
		ev = err.Error()
	}
	d, _ := json.Marshal(WSErrRes{Error: ev})
	return Payload{
		Type: errType,
		Data: d,
	}
}

var (
	ErrQueueFull = errors.New("Send queue full")
)

type PayloadHandler func(payload Payload) error

type Handlers map[Type]PayloadHandler

type Type int

const (
	OKType Type = iota
	ErrType
	Sup

	// Server <-> Server events
	SrvStart
	SrvStop
	SrvRestart
	SrvCopy
	SrvInstall
	SrvUninstall
	SrvLogRaw

	// Server <-> Web Client
	AuthType
	AuthFailType
	AuthOKType
	LogType
	LogQueryOpts
	LogQueryResults
)

type Payload struct {
	Type Type            `json:"payload_type"`
	Data json.RawMessage `json:"data"`
}

// Encode will return an encoded payload suitable for transmission over the wire
func Encode(t Type, p interface{}) ([]byte, error) {
	b, e1 := json.Marshal(p)
	if e1 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload base payload")
	}
	f, e2 := json.Marshal(Payload{
		Type: t,
		Data: b,
	})
	if e2 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload sub payload")
	}
	return f, nil
}

// LogPayload is the container for log/message payloads
type LogPayload struct {
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
}

type Ping struct {
	Nonce int64
}
