package sourcemod

import (
	"errors"
	"sync"
	"time"
)

var ErrReqTooSoon = errors.New("⏱️ request is not available yet")

type seedRequest struct {
	userID    string
	createdOn time.Time
}

// SeedQueue is responsible for keeping track of users who use the !seed command. Servers are only
// Allowed to send a seed request once every 5 minutes by default.
type SeedQueue struct {
	servers map[int]seedRequest
	minTime time.Duration
	mu      *sync.Mutex
}

func NewSeedQueue() SeedQueue {
	return SeedQueue{
		servers: map[int]seedRequest{},
		minTime: time.Second * 300,
		mu:      &sync.Mutex{},
	}
}

func (q *SeedQueue) Allowed(serverID int, userID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	req, found := q.servers[serverID]
	if !found {
		// Check if the user has sent anything on other servers to prevent them being able to
		// cycle through servers spamming the command.
		for _, existingRequest := range q.servers {
			if existingRequest.userID == userID {
				req = existingRequest

				break
			}
		}
	}

	// Check if found request is expired.
	if req.userID == "" || time.Since(req.createdOn) > q.minTime {
		q.servers[serverID] = seedRequest{
			userID:    userID,
			createdOn: time.Now(),
		}

		return true
	}

	return false
}
