package oauth

import (
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type activeState struct {
	State   string
	Created time.Time
}
type LoginStateTracker struct {
	stateMap   map[steamid.SteamID]activeState
	stateMapMu *sync.RWMutex
}

func NewLoginStateTracker() *LoginStateTracker {
	return &LoginStateTracker{
		stateMap:   map[steamid.SteamID]activeState{},
		stateMapMu: &sync.RWMutex{},
	}
}

func (t LoginStateTracker) Get(state string) (steamid.SteamID, bool) {
	t.RemoveExpired()

	t.stateMapMu.Lock()
	defer t.stateMapMu.Unlock()

	var sid steamid.SteamID

	for k, v := range t.stateMap {
		if v.State == state {
			sid = k

			break
		}
	}

	// Only one lookup allowed
	delete(t.stateMap, sid)

	return sid, sid.Valid()
}

func (t LoginStateTracker) Create(steamID steamid.SteamID) string {
	state := stringutil.SecureRandomString(24)

	t.stateMapMu.Lock()
	t.stateMap[steamID] = activeState{
		State:   state,
		Created: time.Now(),
	}
	t.stateMapMu.Unlock()

	return state
}

func (t LoginStateTracker) RemoveExpired() {
	t.stateMapMu.Lock()
	defer t.stateMapMu.Unlock()

	for k := range t.stateMap {
		if time.Since(t.stateMap[k].Created) > time.Second*120 {
			delete(t.stateMap, k)

			continue
		}
	}
}
