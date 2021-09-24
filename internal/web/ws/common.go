package ws

import (
	"encoding/json"
	"github.com/pkg/errors"
)

var (
	ErrQueueFull = errors.New("Send queue full")
)

type PayloadHandler func(payload Payload) error

type Handlers map[Type]PayloadHandler

type Type int

const (
	OKType Type = iota
	ErrType
	Sup

	// Server <-> Server events
	SrvStart
	SrvStop
	SrvRestart
	SrvCopy
	SrvInstall
	SrvUninstall
	SrvLogRaw

	// Server <-> Web Client
	AuthType
	AuthFailType
	AuthOKType
	LogType
	LogQueryOpts
	LogQueryResults
)

type Payload struct {
	Type Type            `json:"payload_type"`
	Data json.RawMessage `json:"data"`
}

// Encode will return an encoded payload suitable for transmission over the wire
func Encode(t Type, p interface{}) ([]byte, error) {
	b, e1 := json.Marshal(p)
	if e1 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload base payload")
	}
	f, e2 := json.Marshal(Payload{
		Type: t,
		Data: b,
	})
	if e2 != nil {
		return nil, errors.Wrapf(e1, "failed to EncodeWSPayload sub payload")
	}
	return f, nil
}
