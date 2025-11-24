package sourcemod

import (
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type seedRequest struct {
	SteamID   steamid.SteamID `json:"steamid"`
	CreatedOn time.Time       `json:"created_on"`
}

// seedQueue is responsible for keeping track of users who use the !seed command. Servers are only
// allowed to send a seed request once every 5 minutes by default.
type seedQueue struct {
	servers map[int]seedRequest
	minTime time.Duration
	mu      *sync.Mutex
}

func (q *seedQueue) allowed(serverID int, steamID steamid.SteamID) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	req, found := q.servers[serverID]
	if !found {
		q.servers[serverID] = seedRequest{
			SteamID:   steamID,
			CreatedOn: time.Now(),
		}

		return true
	}

	if time.Since(req.CreatedOn) > q.minTime {
		delete(q.servers, serverID)

		return true
	}

	return false
}
