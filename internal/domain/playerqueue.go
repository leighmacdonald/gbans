package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type PlayerqueueRepository interface {
	Save(ctx context.Context, message ChatLog) (ChatLog, error)
	Query(ctx context.Context, query PlayerqueueQueryOpts) ([]ChatLog, error)
	Delete(ctx context.Context, messageID ...int64) error
	Message(ctx context.Context, messageID int64) (ChatLog, error)
}

type PlayerqueueUsecase interface {
	AddMessage(ctx context.Context, bodyMD string, user UserProfile) error
	Recent(ctx context.Context, limit uint64) ([]ChatLog, error)
	SetChatStatus(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID, status ChatStatus, reason string) error
	Purge(ctx context.Context, authorID steamid.SteamID, messageID int64, count int) error
	Message(ctx context.Context, messageID int64) (ChatLog, error)
	Connect(ctx context.Context, user UserProfile, conn *websocket.Conn) QueueClient
	Disconnect(client QueueClient)
	JoinLobbies(client QueueClient, servers []int) error
	LeaveLobbies(client QueueClient, servers []int) error
}

type PlayerqueueQueryOpts struct {
	QueryFilter
}

type ChatLog struct {
	MessageID       int64           `json:"message_id"`
	SteamID         steamid.SteamID `json:"steam_id"`
	CreatedOn       time.Time       `json:"created_on"`
	Personaname     string          `json:"personaname"`
	Avatarhash      string          `json:"avatarhash"`
	PermissionLevel int             `json:"permission_level"`
	BodyMD          string          `json:"body_md"`
	Deleted         bool            `json:"deleted"`
}

type QueueClient interface {
	// ID generates a unique identifier for the client connection instance
	ID() string
	// Next handles the incoming operation request
	Next(r *Request) error
	// Ping performs a ping/pong relay with the client
	Ping()
	SteamID() steamid.SteamID
	Name() string
	Avatarhash() string
	// Close disconnects the underlying connection
	Close()
	// Start begins the clients response sender worker
	Start(ctx context.Context)
	Send(response Response)
	// IsTimedOut checks the last ping time to see if a client has stopped pinging us for some reason.
	IsTimedOut() bool
	// HasMessageAccess checks if the user has at leasat readonly access to chat logs
	HasMessageAccess() bool
	// Limit slows down incoming messages, similar to "slow mode", but much dumber, for now.
	Limit()
}

type ChatStatus string

const (
	Readwrite ChatStatus = "readwrite"
	Readonly  ChatStatus = "readonly"
	Noaccess  ChatStatus = "noaccess"
)

type Op int

const (
	// Ping is how you both Join the swarm, and stay in it.
	Ping Op = iota
	Pong
	JoinQueue
	LeaveQueue
	Message
	StateUpdate
	StartGame
	Purge
	Bye
	ChatStatusChange
)

type Request struct {
	Op      Op              `json:"op"`
	Payload json.RawMessage `json:"payload"`
}

type Response struct {
	Op      Op  `json:"op"`
	Payload any `json:"payload"`
}

type ChatStatusChangePayload struct {
	Status ChatStatus `json:"status"`
	Reason string     `json:"reason"`
}
