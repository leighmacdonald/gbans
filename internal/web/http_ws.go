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
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
	"time"
)

// webSocketClient represents the state of a client connected via websockets
type webSocketClient struct {
	Authenticated bool
	Person        *model.Person
	BroadcastLog  bool
	open          bool
	LogFilters    []logparse.MsgType
	ctx           context.Context
	eventChan     chan model.LogEvent
	session       *melody.Session
}

func (c *webSocketClient) reader() {
	for {
		select {
		case e := <-c.eventChan:
			b, _ := json.Marshal(e)
			if err := c.session.Write(b); err != nil {
				log.Errorf("Failed to write to ws")
			}
		case <-c.ctx.Done():
			log.Debugf("ws reader() shutdown")
			return
		}
	}
}

// webSocketState holds the global websocket session state and handlers
type webSocketState struct {
	*sync.RWMutex
	ws       *melody.Melody
	sessions map[*melody.Session]*webSocketClient
}

// newWebSocketState allocates and connects all websocket routes and session states
func newWebSocketState() *webSocketState {
	ws := melody.New()
	wss := &webSocketState{
		RWMutex:  &sync.RWMutex{},
		ws:       ws,
		sessions: map[*melody.Session]*webSocketClient{},
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

type wsAuthReq struct {
	Token string `json:"token"`
}

type wsErrRes struct {
	Error string `json:"error"`
}

func newWSErr(err error) []byte {
	e := wsErrRes{Error: err.Error()}
	b, _ := json.Marshal(e)
	return b
}

func (ws *webSocketState) onMessage(session *melody.Session, msg []byte) {
	ws.Lock()
	defer ws.Unlock()
	c, found := ws.sessions[session]
	if !found {
		log.Errorf("Unknown ws client sent message")
		return
	}
	if !c.Authenticated {
		var w wsAuthReq
		if err := json.Unmarshal(msg, &w); err != nil {
			_ = session.Write(newWSErr(consts.ErrAuthhentication))
		}
		sid, err := sid64FromJWTToken(w.Token)
		if err != nil {
			newWSErr(consts.ErrAuthhentication)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		p, errP := store.GetPersonBySteamID(ctx, sid)
		if errP != nil || p.PermissionLevel < model.PModerator {
			newWSErr(consts.ErrAuthhentication)
			return
		}
		c.Person = p
		log.Debugf("WS User authhenticated successfully")
	} else {
		log.Warnf("WS Unhandled: %v", msg)
	}
}

func (ws *webSocketState) onWSConnect(session *melody.Session) {
	client := &webSocketClient{
		ctx:       context.Background(),
		eventChan: make(chan model.LogEvent),
		open:      true,
		session:   session,
	}
	if err := event.RegisterConsumer(client.eventChan, []logparse.MsgType{logparse.Any}); err != nil {
		log.Warnf("Error registering discord log event reader")
	}
	go client.reader()
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client connect")
	ws.Lock()
	ws.sessions[session] = client
	ws.Unlock()
}

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
