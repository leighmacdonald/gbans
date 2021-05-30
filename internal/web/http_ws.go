package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
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
	LogFilters    []logparse.MsgType
	ctx           context.Context
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
		go func(client *webSocketClient, s *melody.Session) {
			servers, _ := store.GetServers(client.ctx)
			t := time.NewTicker(time.Millisecond * 500)
			logid := int64(0)
			for {
				select {
				case <-t.C:
					sl := model.ServerLog{
						LogID:     logid,
						ServerID:  servers[0].ServerID,
						EventType: logparse.Say,
						Payload: logparse.SayEvt{
							EmptyEvt: logparse.EmptyEvt{},
							SourcePlayer: logparse.SourcePlayer{
								Name: "Test Player",
								PID:  4,
								SID:  76561197961279983,
								Team: logparse.BLU,
							},
							Msg: fmt.Sprintf("This is a test #%d", logid),
						},
						SourceID:  76561197961279983,
						TargetID:  0,
						CreatedOn: config.Now(),
					}
					b, _ := json.Marshal(sl)
					if e := s.Write(b); e != nil {
						log.Errorf("Failed to write ws message: %v", e)
						return
					}

				case <-c.ctx.Done():
					return
				}
				logid++
			}
		}(c, session)
	} else {
		log.Warnf("WS Unhandled: %v", msg)
	}
}

func (ws *webSocketState) onWSConnect(session *melody.Session) {
	ws.Lock()
	defer ws.Unlock()
	ws.sessions[session] = &webSocketClient{
		ctx: context.Background(),
	}
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client connect")
}

func (ws *webSocketState) onWSDisconnect(session *melody.Session) {
	ws.Lock()
	defer ws.Unlock()
	delete(ws.sessions, session)
	log.WithField("addr", session.Request.RemoteAddr).Infof("WS client disconnect")
}
