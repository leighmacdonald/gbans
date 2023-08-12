package app

import (
	"sync"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
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
	allEventsMu    *sync.RWMutex
}

// Consume will register a channel to receive new log events as they come in. If no event keys
// are provided, all events will be sent.
func (eb *eventBroadcaster[k, v]) Consume(serverEventChan chan v, keys ...k) error {
	if len(keys) > 0 {
		eb.eventReadersMu.Lock()
		for _, msgType := range keys {
			_, found := eb.eventReaders[msgType]
			if !found {
				eb.eventReaders[msgType] = []chan v{}
			}

			eb.eventReaders[msgType] = append(eb.eventReaders[msgType], serverEventChan)
		}
		eb.eventReadersMu.Unlock()
	} else {
		eb.allEventsMu.Lock()
		for _, existing := range eb.allEvents {
			if existing == serverEventChan {
				eb.allEventsMu.Unlock()

				return errors.New("Duplicate channel registration")
			}
		}

		eb.allEvents = append(eb.allEvents, serverEventChan)
		eb.allEventsMu.Unlock()
	}

	return nil
}

// Emit is used to send out events to and registered reader channels.
func (eb *eventBroadcaster[k, v]) Emit(et k, event v) {
	// Ensure we also send to Any handlers for all events.
	for _, eventType := range []k{et} {
		eb.allEventsMu.RLock()
		eb.eventReadersMu.RLock()
		specificReaders, specificReadersFound := eb.eventReaders[eventType]

		readerChannels := eb.allEvents

		if specificReadersFound {
			readerChannels = append(readerChannels, specificReaders...)
		}

		for _, reader := range readerChannels {
			reader <- event
		}

		eb.eventReadersMu.RUnlock()
		eb.allEventsMu.RUnlock()
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
