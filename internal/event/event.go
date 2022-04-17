// Package event implements an event dispatcher for incoming log events. It sends the
// messages to the registered matching event reader channels that have been registered for the
// event type.
package event

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	log "github.com/sirupsen/logrus"
	"sync"
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

// RegisterConsumer will register a channel to receive new log events as they come in
func RegisterConsumer(serverEventChan chan model.ServerEvent, msgTypes []logparse.EventType) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for _, msgType := range msgTypes {
		_, found := logEventReaders[msgType]
		if !found {
			logEventReaders[msgType] = []chan model.ServerEvent{}
		}
		logEventReaders[msgType] = append(logEventReaders[msgType], serverEventChan)
	}
	log.WithFields(log.Fields{"count": len(msgTypes)}).Trace("Registered event reader(s)")
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
		for rt, reader := range readers {
			select {
			case reader <- serverEvent:
			default:
				log.WithFields(log.Fields{"type": rt}).Errorf("Failed to write to log event channel")
			}

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
