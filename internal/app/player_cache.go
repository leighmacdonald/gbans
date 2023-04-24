package app

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"sync"
	"time"
)

type playerEventState struct {
	team      logparse.Team
	class     logparse.PlayerClass
	updatedAt time.Time
}

type playerCache struct {
	*sync.RWMutex
	logger *zap.Logger
	state  map[steamid.SID64]playerEventState
}

func newPlayerCache(logger *zap.Logger) *playerCache {
	pc := playerCache{
		RWMutex: &sync.RWMutex{},
		logger:  logger.Named("player_cache"),
		state:   map[steamid.SID64]playerEventState{},
	}
	go pc.cleanupWorker()
	return &pc
}

func (cache *playerCache) setTeam(sid steamid.SID64, team logparse.Team) {
	cache.Lock()
	defer cache.Unlock()
	state, found := cache.state[sid]
	if !found {
		state = playerEventState{}
	}
	state.team = team
	state.updatedAt = config.Now()
	cache.state[sid] = state
}

func (cache *playerCache) setClass(sid steamid.SID64, class logparse.PlayerClass) {
	cache.Lock()
	defer cache.Unlock()
	state, found := cache.state[sid]
	if !found {
		state = playerEventState{}
	}
	state.class = class
	state.updatedAt = config.Now()
	cache.state[sid] = state
}

func (cache *playerCache) getClass(sid steamid.SID64) logparse.PlayerClass {
	cache.RLock()
	defer cache.RUnlock()
	state, found := cache.state[sid]
	if !found {
		return logparse.Spectator
	}
	return state.class
}

func (cache *playerCache) getTeam(sid steamid.SID64) logparse.Team {
	cache.RLock()
	defer cache.RUnlock()
	state, found := cache.state[sid]
	if !found {
		return logparse.SPEC
	}
	return state.team
}

func (cache *playerCache) cleanupWorker() {
	ticker := time.NewTicker(20 * time.Second)
	for {
		<-ticker.C
		now := config.Now()
		cache.Lock()
		for steamId, state := range cache.state {
			if now.Sub(state.updatedAt) > time.Hour {
				delete(cache.state, steamId)
				cache.logger.Debug("Player cache expired", zap.Int64("sid64", steamId.Int64()))
			}
		}
		cache.Unlock()
	}
}
