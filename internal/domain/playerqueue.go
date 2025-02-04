package domain

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Message struct {
	MessageID       uuid.UUID `json:"message_id"`
	SteamID         string    `json:"steam_id"`
	CreatedOn       time.Time `json:"created_on"`
	Personaname     string    `json:"personaname"`
	Avatarhash      string    `json:"avatarhash"`
	PermissionLevel Privilege `json:"permission_level"`
	BodyMD          string    `json:"body_md"`
	Deleted         bool      `json:"deleted"`
}

type PlayerqueueRepository interface {
	Save(ctx context.Context, message Message) (Message, error)
	Query(ctx context.Context, query PlayerqueueQueryOpts) ([]Message, error)
	Delete(ctx context.Context, messageID ...uuid.UUID) error
}

type PlayerqueueUsecase interface {
	Add(ctx context.Context, message Message) (Message, error)
	Recent(ctx context.Context, limit uint64) ([]Message, error)
	SetChatStatus(ctx context.Context, steamID steamid.SteamID, status ChatStatus) error
	Delete(ctx context.Context, messageID ...uuid.UUID) error
}

type PlayerqueueQueryOpts struct {
	QueryFilter
}
