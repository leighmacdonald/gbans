package fp

import (
	"sync"

	"github.com/pkg/errors"
)

type Broadcaster[T comparable, V any] struct {
	// Each log event can have any number of channels associated with them
	// Events are sent to all channels in a fan-out style.
	readerMap    map[T][]chan V
	allReaders   []chan V
	readerMapMu  *sync.RWMutex
	allReadersMu *sync.RWMutex
}

func NewBroadcaster[T comparable, V any]() *Broadcaster[T, V] {
	return &Broadcaster[T, V]{
		readerMap:    map[T][]chan V{},
		readerMapMu:  &sync.RWMutex{},
		allReadersMu: &sync.RWMutex{},
	}
}

// Consume will register a channel to receive new log events as they come in. If no event keys
// are provided, all events will be sent.
func (eb *Broadcaster[k, v]) Consume(serverEventChan chan v, keys ...k) error {
	if len(keys) > 0 {
		eb.readerMapMu.Lock()
		for _, msgType := range keys {
			_, found := eb.readerMap[msgType]
			if !found {
				eb.readerMap[msgType] = []chan v{}
			}

			eb.readerMap[msgType] = append(eb.readerMap[msgType], serverEventChan)
		}
		eb.readerMapMu.Unlock()
	} else {
		eb.allReadersMu.Lock()
		for _, existing := range eb.allReaders {
			if existing == serverEventChan {
				eb.allReadersMu.Unlock()

				return errors.New("Duplicate channel registration")
			}
		}

		eb.allReaders = append(eb.allReaders, serverEventChan)
		eb.allReadersMu.Unlock()
	}

	return nil
}

// Emit is used to send out events to all registered reader channels.
func (eb *Broadcaster[k, v]) Emit(key k, value v) {
	// Ensure we also send to Any handlers for all events.
	for _, eventType := range []k{key} {
		eb.allReadersMu.RLock()
		eb.readerMapMu.RLock()
		specificReaders, specificReadersFound := eb.readerMap[eventType]

		readerChannels := eb.allReaders

		if specificReadersFound {
			readerChannels = append(readerChannels, specificReaders...)
		}

		for _, reader := range readerChannels {
			reader <- value
		}

		eb.readerMapMu.RUnlock()
		eb.allReadersMu.RUnlock()
	}
}

func (eb *Broadcaster[k, v]) removeChan(channels []chan v, eventChan chan v) []chan v {
	var newChannels []chan v

	for _, channel := range channels {
		if channel != eventChan {
			newChannels = append(newChannels, channel)
		}
	}

	return newChannels
}

// Unregister will remove the channel from any matching event readers.
func (eb *Broadcaster[k, v]) Unregister(value chan v) error {
	eb.readerMapMu.Lock()

	for eType, eventReaders := range eb.readerMap {
		eb.readerMap[eType] = eb.removeChan(eventReaders, value)
	}

	eb.readerMapMu.Unlock()

	eb.allReadersMu.Lock()
	eb.allReaders = eb.removeChan(eb.allReaders, value)
	eb.allReadersMu.Unlock()

	return nil
}
