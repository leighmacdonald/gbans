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

type eventInterface interface {
	serverEvent
}

type eventBroadcaster[T comparable, V eventInterface] struct {
	// Each log event can have any number of channels associated with them
	// Events are sent to all channels in a fan-out style.
	eventReaders   map[T][]chan V
	allEvents      []chan V
	eventReadersMu *sync.RWMutex
}

// Consume will register a channel to receive new log events as they come in.
func (eb *eventBroadcaster[k, v]) Consume(serverEventChan chan v, msgTypes []k) error {
	eb.eventReadersMu.Lock()
	defer eb.eventReadersMu.Unlock()

	for _, msgType := range msgTypes {
		_, found := eb.eventReaders[msgType]
		if !found {
			eb.eventReaders[msgType] = []chan v{}
		}

		eb.eventReaders[msgType] = append(eb.eventReaders[msgType], serverEventChan)
	}

	return nil
}

// Emit is used to send out events to and registered reader channels.
func (eb *eventBroadcaster[k, v]) Emit(et k, event v) {
	// Ensure we also send to Any handlers for all events.
	for _, eventType := range []k{et} {
		eb.eventReadersMu.RLock()
		specificReaders, specificReadersFound := eb.eventReaders[eventType]
		eb.eventReadersMu.RUnlock()

		readerChannels := eb.allEvents
		if specificReadersFound {
			readerChannels = append(readerChannels, specificReaders...)
		}

		for _, reader := range readerChannels {
			reader <- event
		}
	}
}

func (eb *eventBroadcaster[k, v]) removeChan(channels []chan v, eventChan chan v) []chan v {
	var newChannels []chan v

	for _, channel := range channels {
		if channel != eventChan {
			newChannels = append(newChannels, channel)
		}
	}

	return newChannels
}

// UnregisterConsumer will remove the channel from any matching event readers.
func (eb *eventBroadcaster[k, v]) UnregisterConsumer(value chan v) error {
	eb.eventReadersMu.Lock()
	defer eb.eventReadersMu.Unlock()

	for eType, eventReaders := range eb.eventReaders {
		eb.eventReaders[eType] = eb.removeChan(eventReaders, value)
	}

	return nil
}
