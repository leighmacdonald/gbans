package playerqueue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type PlayerqueueRepository interface {
	Save(ctx context.Context, message ChatLog) (ChatLog, error)
	Query(ctx context.Context, query PlayerqueueQueryOpts) ([]ChatLog, error)
	Delete(ctx context.Context, messageID ...int64) error
	Message(ctx context.Context, messageID int64) (ChatLog, error)
}

type PlayerqueueUsecase interface {
	AddMessage(ctx context.Context, bodyMD string, user person.UserProfile) error
	Recent(ctx context.Context, limit uint64) ([]ChatLog, error)
	SetChatStatus(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID, status ChatStatus, reason string) error
	Purge(ctx context.Context, authorID steamid.SteamID, messageID int64, count int) error
	Message(ctx context.Context, messageID int64) (ChatLog, error)
	Connect(ctx context.Context, user person.UserProfile, conn *websocket.Conn) QueueClient
	Disconnect(client QueueClient)
	JoinLobbies(client QueueClient, servers []int) error
	LeaveLobbies(client QueueClient, servers []int) error
	Start(ctx context.Context)
}

type PlayerqueueQueryOpts struct {
	domain.QueryFilter
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
	SteamID() steamid.SteamID
	Name() string
	Avatarhash() string
	// Close disconnects the underlying connection
	Close()
	// Start begins the clients response sender worker
	Start(ctx context.Context)
	Send(response Response)
	// HasMessageAccess checks if the user has at least readonly access to chat logs
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
	JoinQueue Op = iota
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
