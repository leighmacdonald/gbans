package domain

import (
	"encoding/json"
	"github.com/gofrs/uuid/v5"
	"time"
)

type ServerQueueOperation = int

const (
	// Ping is how you both join the swarm, and stay in it.
	Ping ServerQueueOperation = iota
	Pong
	Join
	Leave
	MessageSend
	MessageRecv
	StateUpdate
)

type PingPayload struct {
	CreatedOn time.Time `json:"created_on"`
}

type PongPayload = PingPayload

type HelloPayload struct {
	BodyMD string `json:"body_md"`
}

type GoodbyePayload struct {
	BodyMD string `json:"body_md"`
}

type JoinPayload struct {
	Servers []int `json:"servers"`
}

type LeavePayload struct {
	Servers []int `json:"serverds"`
}

type ServerQueueMessage struct {
	ID              uuid.UUID `json:"id"`
	SteamID         string    `json:"steam_id"`
	CreatedOn       time.Time `json:"created_on"`
	Personaname     string    `json:"personaname"`
	Avatarhash      string    `json:"avatarhash"`
	PermissionLevel Privilege `json:"permission_level"`
	BodyMD          string    `json:"body_md"`
}

type ServerQueuePayloadInbound struct {
	Op      ServerQueueOperation `json:"op"`
	Payload json.RawMessage      `json:"payload"`
}
type ServerQueuePayloadOutbound struct {
	Op      ServerQueueOperation `json:"op"`
	Payload any                  `json:"payload"`
}
