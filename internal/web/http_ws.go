package web

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	log "github.com/sirupsen/logrus"
	"gopkg.in/olahol/melody.v1"
	"sync"
)

// webSocketClient represents the state of a client connected via websockets
type webSocketClient struct {
	BroadcastLog bool
	LogFilters   []logparse.MsgType
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

func (ws *webSocketState) onWSConnect(session *melody.Session) {
	ws.Lock()
	ws.Unlock()
	ws.sessions[session] = &webSocketClient{}
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client connect")
}

func (ws *webSocketState) onWSDisconnect(session *melody.Session) {
	ws.Lock()
	defer ws.Unlock()
	delete(ws.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
}
