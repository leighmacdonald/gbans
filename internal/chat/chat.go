package chat

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ChatRepository interface {
	GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error)
	Start(ctx context.Context)
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error)
	GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
	GetWarningChan() chan NewUserWarning
}

type ChatUsecase interface {
	Start(ctx context.Context)
	WarningState() map[string][]UserWarning
	GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error)
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, user domain.PersonInfo, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error)
	GetPersonMessageContext(ctx context.Context, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
}

type ChatHistoryQueryFilter struct {
	domain.QueryFilter
	domain.SourceIDField
	Personaname   string     `json:"personaname,omitempty"`
	ServerID      int        `json:"server_id,omitempty"`
	DateStart     *time.Time `json:"date_start,omitempty"`
	DateEnd       *time.Time `json:"date_end,omitempty"`
	Unrestricted  bool       `json:"-"`
	DontCalcTotal bool       `json:"-"`
	FlaggedOnly   bool       `json:"flagged_only"`
}

func (f ChatHistoryQueryFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SourceID)

	return sid, sid.Valid()
}

type TopChatterResult struct {
	Name    string
	SteamID steamid.SteamID
	Count   int
}

type PersonMessage struct {
	PersonMessageID   int64           `json:"person_message_id"`
	MatchID           uuid.UUID       `json:"match_id"`
	SteamID           steamid.SteamID `json:"steam_id"`
	AvatarHash        string          `json:"avatar_hash"`
	PersonaName       string          `json:"persona_name"`
	ServerName        string          `json:"server_name"`
	ServerID          int             `json:"server_id"`
	Body              string          `json:"body"`
	Team              bool            `json:"team"`
	CreatedOn         time.Time       `json:"created_on"`
	AutoFilterFlagged int64           `json:"auto_filter_flagged"`
}

type PersonMessages []PersonMessage

type QueryChatHistoryResult struct {
	PersonMessage
	Pattern string `json:"pattern"`
}
