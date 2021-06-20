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
	logEventReaders   map[logparse.MsgType][]chan model.ServerEvent
	logEventReadersMu *sync.RWMutex
)

func init() {
	logEventReaders = map[logparse.MsgType][]chan model.ServerEvent{}
	logEventReadersMu = &sync.RWMutex{}
}

// RegisterConsumer will register a channel to receive new log events as they come in
func RegisterConsumer(r chan model.ServerEvent, msgTypes []logparse.MsgType) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for _, msgType := range msgTypes {
		_, found := logEventReaders[msgType]
		if !found {
			logEventReaders[msgType] = []chan model.ServerEvent{}
		}
		logEventReaders[msgType] = append(logEventReaders[msgType], r)
	}
	log.Debugf("Registered %d event readers", len(msgTypes))
	return nil
}

func Emit(le model.ServerEvent) {
	// Ensure we also send to Any handlers for all events.
	logEventReadersMu.RLock()
	defer logEventReadersMu.RUnlock()
	for _, typ := range []logparse.MsgType{le.EventType, logparse.Any} {
		readers, ok := logEventReaders[typ]
		if !ok {
			continue
		}
		for _, reader := range readers {
			reader <- le
		}
	}
}

func removeChan(channels []chan model.ServerEvent, c chan model.ServerEvent) []chan model.ServerEvent {
	var newChannels []chan model.ServerEvent
	for _, i := range channels {
		if i != c {
			newChannels = append(newChannels, i)
		}
	}
	return newChannels
}

func UnregisterConsumer(r chan model.ServerEvent) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for k, v := range logEventReaders {
		logEventReaders[k] = removeChan(v, r)
	}
	return nil
}
