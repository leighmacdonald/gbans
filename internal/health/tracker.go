package health

import (
	"maps"
	"sync"
	"time"
)

// Event records the last outcome of a health-relevant operation. Err is empty
// when the last operation succeeded.
type Event struct {
	At  time.Time
	Err string
}

// Tracker holds in-memory health state updated by the rest of the application.
// All access is mutex protected so checks can be recorded from any goroutine.
type Tracker struct {
	mu      sync.RWMutex
	started time.Time
	rcon    map[int32]Event
	ssh     map[string]Event
	events  map[string]Event
}

func NewTracker() *Tracker {
	return &Tracker{
		started: time.Now(),
		rcon:    make(map[int32]Event),
		ssh:     make(map[string]Event),
		events:  make(map[string]Event),
	}
}

func recordEvent(events map[string]Event, name string, err error) {
	entry := Event{At: time.Now()}
	if err != nil {
		entry.Err = err.Error()
	}
	events[name] = entry
}

// Record stores a named event outcome (stats refresh, worker tick, etc.).
func (t *Tracker) Record(name string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	recordEvent(t.events, name, err)
}

// RecordRCON records the outcome of an RCON command for one server.
func (t *Tracker) RecordRCON(serverID int32, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry := Event{At: time.Now()}
	if err != nil {
		entry.Err = err.Error()
	}
	t.rcon[serverID] = entry
}

// RecordSSH records the outcome of an SSH/SCP connection for one host.
func (t *Tracker) RecordSSH(host string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	recordEvent(t.ssh, host, err)
}

// Started returns the process start time.
func (t *Tracker) Started() time.Time {
	return t.started
}

// Snapshot returns a copy of all recorded state.
func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snapshot := Snapshot{
		Started: t.started,
		RCON:    make(map[int32]Event, len(t.rcon)),
		SSH:     make(map[string]Event, len(t.ssh)),
		Events:  make(map[string]Event, len(t.events)),
	}
	maps.Copy(snapshot.RCON, t.rcon)
	maps.Copy(snapshot.SSH, t.ssh)
	maps.Copy(snapshot.Events, t.events)

	return snapshot
}

// Snapshot is an immutable copy of the tracker state.
type Snapshot struct {
	Started time.Time
	RCON    map[int32]Event
	SSH     map[string]Event
	Events  map[string]Event
}

// ServerSnapshot is the per-server state the health check exposes.
type ServerSnapshot struct {
	ServerID         int32
	Name             string
	LastStatusUpdate time.Time
	TokenCreatedOn   time.Time
	STVPort          uint16
	Enabled          bool
}
