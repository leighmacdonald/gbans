package state

import (
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/rumblefrog/go-a2s"
	"sync"
)

var (
	serverStateMu *sync.RWMutex
	serverStates  map[string]*ServerState
)

type gameType string

const (
	// unknown gameType = "Unknown"
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

func init() {
	serverStates = make(map[string]*ServerState)
	serverStateMu = &sync.RWMutex{}
}
