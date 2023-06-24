package app

import (
	"sync"

	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type eventBroadcaster struct {
	// Each log event can have any number of channels associated with them
	// Events are sent to all channels in a fan-out style.
	logEventReaders   map[logparse.EventType][]chan model.ServerEvent
	logEventReadersMu *sync.RWMutex
}

func newEventBroadcaster() *eventBroadcaster {
	return &eventBroadcaster{
		logEventReaders:   map[logparse.EventType][]chan model.ServerEvent{},
		logEventReadersMu: &sync.RWMutex{},
	}
}

// Consume will register a channel to receive new log events as they come in.
func (eb *eventBroadcaster) Consume(serverEventChan chan model.ServerEvent, msgTypes []logparse.EventType) error {
	eb.logEventReadersMu.Lock()
	defer eb.logEventReadersMu.Unlock()
	for _, msgType := range msgTypes {
		_, found := eb.logEventReaders[msgType]
		if !found {
			eb.logEventReaders[msgType] = []chan model.ServerEvent{}
		}
		eb.logEventReaders[msgType] = append(eb.logEventReaders[msgType], serverEventChan)
	}
	return nil
}

// Emit is used to send out events to and registered reader channels.
func (eb *eventBroadcaster) Emit(serverEvent model.ServerEvent) {
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

func (eb *eventBroadcaster) removeChan(channels []chan model.ServerEvent, serverEventChan chan model.ServerEvent) []chan model.ServerEvent {
	var newChannels []chan model.ServerEvent
	for _, channel := range channels {
		if channel != serverEventChan {
			newChannels = append(newChannels, channel)
		}
	}
	return newChannels
}

// UnregisterConsumer will remove the channel from any matching event readers.
func (eb *eventBroadcaster) UnregisterConsumer(serverEventChan chan model.ServerEvent) error {
	eb.logEventReadersMu.Lock()
	defer eb.logEventReadersMu.Unlock()
	for eType, eventReaders := range eb.logEventReaders {
		eb.logEventReaders[eType] = eb.removeChan(eventReaders, serverEventChan)
	}
	return nil
}
