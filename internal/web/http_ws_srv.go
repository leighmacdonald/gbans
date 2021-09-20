package web

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
)

type ServerPayloadType int

const (
	SrvStart ServerPayloadType = iota
	SrvStop
	SrvRestart
	SrvCopy
	SrvInstall
	SrvUninstall
	SrvLogRaw
)

type ServerSocketPayload struct {
	PayloadType ServerPayloadType `json:"payload_type"`
	Data        json.RawMessage   `json:"data"`
}

type RPCClient struct {
	addr   string
	l      log.Logger
	done   chan interface{}
	writer chan ServerSocketPayload
}

func NewRPCClient(addr string) (*RPCClient, error) {
	return &RPCClient{addr: addr, done: make(chan interface{}, 100)}, nil
}

func (a *RPCClient) Stop() {
	close(a.done)
}

func (a *RPCClient) Send(pt ServerPayloadType, payload interface{}) error {
	p, errEnc := json.Marshal(payload)
	if errEnc != nil {
		return errEnc
	}
	select {
	case a.writer <- ServerSocketPayload{PayloadType: pt, Data: p}:
	default:
		return errors.New("Send queue full")
	}
	return nil
}

type RPCStartCommand struct{}
type RPCStopCommand struct{}
type RPCRestartCommand struct{}
type RPCCopyCommand struct{}
type RPCInstallCommand struct{}
type RPCUnstallCommand struct{}

func (a *RPCClient) OnInstallCommand(c RPCInstallCommand) {

}
func (a *RPCClient) OnUninstallCommand(c RPCUnstallCommand) {

}
func (a *RPCClient) OnCopyCommand(c RPCCopyCommand) {

}

func (a *RPCClient) OnStartCommand(c RPCStartCommand) {

}

func (a *RPCClient) OnStopCommand(c RPCStopCommand) {

}

func (a *RPCClient) OnRestartCommand(c RPCRestartCommand) {

}

func (a *RPCClient) Start() error {
	a.l.Debugf("Connecting")
	// TODO auth header
	c, _, err := websocket.DefaultDialer.Dial(a.addr, nil)
	if err != nil {
		return err
	}
	defer func() {
		if errC := c.Close(); errC != nil {
			a.l.Error("Failed closing websocket connection cleanly: %v", errC)
		}
	}()
	go func() {
		defer close(a.done)
		for {
			var p ServerSocketPayload
			if e := c.ReadJSON(&p); e != nil {
				a.l.Errorf("Failed to read json from server: %v", e)
			}
			switch p.PayloadType {
			case SrvCopy:
				var d RPCCopyCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnCopyCommand(d)
			case SrvInstall:
				var d RPCInstallCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnInstallCommand(d)
			case SrvUninstall:
				var d RPCUnstallCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnUninstallCommand(d)
			case SrvStart:
				var d RPCStartCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnStartCommand(d)
			case SrvStop:
				var d RPCStopCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnStopCommand(d)
			case SrvRestart:
				var d RPCRestartCommand
				if errDec := json.Unmarshal(p.Data, &d); errDec != nil {
					a.l.Errorf("Failed to decode json payload: %v", errDec)
				}
				a.OnRestartCommand(d)
			}
		}
	}()
	for {
		select {
		case <-a.done:
			return nil
		case pl := <-a.writer:
			if errW := c.WriteJSON(pl); errW != nil {
				return errors.Wrapf(errW, "Failed to write to websocket")
			}
		}
	}
}

// serverSocketState holds the global websocket session state and handlers
type serverSocketState struct {
	*sync.RWMutex
	ws         *melody.Melody
	logMsgChan chan LogPayload
	sessions   map[*melody.Session]*socketSession
}

// newAgentServiceState allocates and connects all websocket routes and session states for
// server agent connections.
func newAgentServiceState() *serverSocketState {
	wsWeb := melody.New()
	wss := &serverSocketState{
		RWMutex:  &sync.RWMutex{},
		ws:       wsWeb,
		sessions: map[*melody.Session]*socketSession{},
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

func (ws *serverSocketState) onWSStart(c *gin.Context) {
	if err := ws.ws.HandleRequest(c.Writer, c.Request); err != nil {
		log.Errorf("Error handling ws request: %v", err)
	}
}

// onMessage handles incoming websocket payloads
// We always return authentication errors until the client is fully authed. This is to prevent
// any leaking of information to an attacker that can be further leveraged to aide in further
// attacks by this or other vectors.
func (ws *serverSocketState) onMessage(session *melody.Session, msg []byte) {
	ws.Lock()
	defer ws.Unlock()
	sockSession, found := ws.sessions[session]
	if !found {
		log.Errorf("Unknown ws client sent message")
		return
	}

	var w ServerSocketPayload
	if err := json.Unmarshal(msg, &w); err != nil {
		sockSession.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal ws payload")
		return
	}

	switch w.PayloadType {
	case SrvLogRaw:
		var l LogPayload
		if err := json.Unmarshal(w.Data, &l); err != nil {
			sockSession.err(ErrType, consts.ErrMalformedRequest, "Failed to unmarshal logpayload data")
			return
		}
		select {
		case ws.logMsgChan <- l:
		default:
			log.Error("Log message channel queue full, message discarded")
		}
	default:
		sockSession.Log().Debugf("Unhandled payload: %v", w)
	}
}

// onWSConnect sets up the websocket client in the session list and registers it to receive all log events
// by default.
func (ws *serverSocketState) onWSConnect(session *melody.Session) {
	client := &socketSession{
		State:     Closed,
		ctx:       context.Background(),
		eventChan: make(chan model.ServerEvent),
		session:   session,
		sendQ:     make(chan []byte, sendQueueSize),
		recvQ:     make(chan []byte, recvQueueSize),
	}
	go client.reader()
	go client.writer()
	client.State = AwaitingAuthentication
	ws.Lock()
	ws.sessions[session] = client
	ws.Unlock()
	client.Log().Infof("WS client connect")

}

// onWSDisconnect will remove the client from the active session list and unregister itself
// from the event broadcasts
func (ws *serverSocketState) onWSDisconnect(session *melody.Session) {
	ws.Lock()
	defer ws.Unlock()
	c, found := ws.sessions[session]
	if !found {
		log.Errorf("Unregistered ws client")
		return
	}
	c.State = Closing
	delete(ws.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
	if err := event.UnregisterConsumer(c.eventChan); err != nil {
		log.Errorf("Failed to unregister event consumer")
	}
	// TODO cleanup remaining queues
	c.State = Closed

}
