package state

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	serverStateMu *sync.RWMutex
	serverStates  map[string]*ServerState
	gameState     map[string]*Game
	gameHistory   []*Game
	handler       = map[logparse.MsgType]eventHandler{
		logparse.LogStart:     onLogStart,
		logparse.Connected:    onConnected,
		logparse.Disconnected: onDisconnected,
		logparse.LogStop:      onLogStop,
	}
)

type eventHandler func(event model.LogEvent)

type gameType string

const (
	//unknown gameType = "Unknown"
	TF2 gameType = "team Fortress 2"
	//cs      gameType = "Counter-Strike"
	//csgo    gameType = "Counter-Strike: Global Offensive"
)

type ServerState struct {
	Addr     string
	Port     int
	Slots    int
	GameType gameType
	A2SInfo  *a2s.ServerInfo
	extra.Status
	// TODO Find better way to track this
	Alive bool
}

func SetServer(name string, state ServerState) {
	serverStateMu.Lock()
	serverStates[name] = &state
	serverStateMu.Unlock()
}

func ServersAlive() int {
	var i int
	serverStateMu.RLock()
	defer serverStateMu.RUnlock()
	for _, server := range serverStates {
		if server.Alive {
			i++
		}
	}
	return i
}

func Start(ctx context.Context, eventChan chan model.LogEvent) {
	for {
		select {
		case e := <-eventChan:
			fn, found := handler[e.Type]
			if !found {
				log.Warnf("Unhandled event")
				continue
			}
			fn(e)
		case <-ctx.Done():
			return
		}
	}
}

func onConnected(evt model.LogEvent) {
	p := newPlayer()
	s, ok := gameState[evt.Server.ServerName]
	if !ok {
		return
	}
	s.Players = append(s.Players, p)
}

func onDisconnected(evt model.LogEvent) {

}

func onLogStop(evt model.LogEvent) {
	game, exists := gameState[evt.Server.ServerName]
	if !exists {
		log.Warnf("Got logStop event for unknown game")
		return
	}
	gameHistory = append(gameHistory, game)
	delete(gameState, evt.Server.ServerName)
}

func onLogStart(evt model.LogEvent) {
	game, exists := gameState[evt.Server.ServerName]
	if exists {
		log.Debugf("Clearing existing game state")
		gameHistory = append(gameHistory, game)
	}
	gameState[evt.Server.ServerName] = newGame()
}

func init() {
	serverStates = make(map[string]*ServerState)
	serverStateMu = &sync.RWMutex{}
	gameState = make(map[string]*Game)
}
