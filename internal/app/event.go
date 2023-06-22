// Package event implements an event dispatcher for incoming log events. It sends the
// messages to the registered matching event reader channels that have been registered for the
// event type.
package app

import (
	"sync"

	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

var (
	// Each log event can have any number of channels associated with them
	// Events are sent to all channels in a fan-out style
	logEventReaders   map[logparse.EventType][]chan model.ServerEvent
	logEventReadersMu *sync.RWMutex
)

func init() {
	logEventReaders = map[logparse.EventType][]chan model.ServerEvent{}
	logEventReadersMu = &sync.RWMutex{}
}

// Consume will register a channel to receive new log events as they come in
func Consume(serverEventChan chan model.ServerEvent, msgTypes []logparse.EventType) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for _, msgType := range msgTypes {
		_, found := logEventReaders[msgType]
		if !found {
			logEventReaders[msgType] = []chan model.ServerEvent{}
		}
		logEventReaders[msgType] = append(logEventReaders[msgType], serverEventChan)
	}
	return nil
}

// Emit is used to send out events to and registered reader channels.
func Emit(serverEvent model.ServerEvent) {
	// Ensure we also send to Any handlers for all events.
	for _, eventType := range []logparse.EventType{serverEvent.EventType, logparse.Any} {
		logEventReadersMu.RLock()
		readers, ok := logEventReaders[eventType]
		logEventReadersMu.RUnlock()
		if !ok {
			continue
		}
		for _, reader := range readers {
			reader <- serverEvent
		}
	}
}

func removeChan(channels []chan model.ServerEvent, serverEventChan chan model.ServerEvent) []chan model.ServerEvent {
	var newChannels []chan model.ServerEvent
	for _, channel := range channels {
		if channel != serverEventChan {
			newChannels = append(newChannels, channel)
		}
	}
	return newChannels
}

// UnregisterConsumer will remove the channel from any matching event readers
func UnregisterConsumer(serverEventChan chan model.ServerEvent) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for eType, eventReaders := range logEventReaders {
		logEventReaders[eType] = removeChan(eventReaders, serverEventChan)
	}
	return nil
}
