package app

import (
	"sync"

	"github.com/leighmacdonald/gbans/pkg/logparse"
)

// serverEvent is a flat struct encapsulating a parsed log event.
type serverEvent struct {
	ServerID   int
	ServerName string
	*logparse.Results
}

type eventBroadcaster struct {
	// Each log event can have any number of channels associated with them
	// Events are sent to all channels in a fan-out style.
	logEventReaders   map[logparse.EventType][]chan serverEvent
	logEventReadersMu *sync.RWMutex
}

func newEventBroadcaster() *eventBroadcaster {
	return &eventBroadcaster{
		logEventReaders:   map[logparse.EventType][]chan serverEvent{},
		logEventReadersMu: &sync.RWMutex{},
	}
}

// Consume will register a channel to receive new log events as they come in.
func (eb *eventBroadcaster) Consume(serverEventChan chan serverEvent, msgTypes []logparse.EventType) error {
	eb.logEventReadersMu.Lock()
	defer eb.logEventReadersMu.Unlock()

	for _, msgType := range msgTypes {
		_, found := eb.logEventReaders[msgType]
		if !found {
			eb.logEventReaders[msgType] = []chan serverEvent{}
		}

		eb.logEventReaders[msgType] = append(eb.logEventReaders[msgType], serverEventChan)
	}

	return nil
}

// Emit is used to send out events to and registered reader channels.
func (eb *eventBroadcaster) Emit(serverEvent serverEvent) {
	// Ensure we also send to Any handlers for all events.
	for _, eventType := range []logparse.EventType{serverEvent.EventType, logparse.Any} {
		eb.logEventReadersMu.RLock()
		readers, ok := eb.logEventReaders[eventType]
		eb.logEventReadersMu.RUnlock()

		if !ok {
			continue
		}

		for _, reader := range readers {
			reader <- serverEvent
		}
	}
}

func (eb *eventBroadcaster) removeChan(channels []chan serverEvent, serverEventChan chan serverEvent) []chan serverEvent {
	var newChannels []chan serverEvent

	for _, channel := range channels {
		if channel != serverEventChan {
			newChannels = append(newChannels, channel)
		}
	}

	return newChannels
}

// UnregisterConsumer will remove the channel from any matching event readers.
func (eb *eventBroadcaster) UnregisterConsumer(serverEventChan chan serverEvent) error {
	eb.logEventReadersMu.Lock()
	defer eb.logEventReadersMu.Unlock()

	for eType, eventReaders := range eb.logEventReaders {
		eb.logEventReaders[eType] = eb.removeChan(eventReaders, serverEventChan)
	}

	return nil
}
