package domain

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type VoteQueryFilter struct {
	QueryFilter
	SourceIDField
	TargetIDField
	ServerID int    `json:"server_id"`
	Name     string `json:"name"`
	Success  int    `json:"success"` // -1 = any, 0 = false, 1 = true
	Code     bool   `json:"code"`
}

type VoteRepository interface {
	Query(ctx context.Context, filter VoteQueryFilter) ([]VoteResult, int64, error)
	AddResult(ctx context.Context, voteResult VoteResult) error
}

type VoteUsecase interface {
	Query(ctx context.Context, filter VoteQueryFilter) ([]VoteResult, int64, error)
	Start(ctx context.Context)
}

type VoteResult struct {
	VoteID           int               `json:"vote_id"`
	SourceID         steamid.SteamID   `json:"source_id"`
	SourceName       string            `json:"source_name"`
	SourceAvatarHash string            `json:"source_avatar_hash"`
	TargetID         steamid.SteamID   `json:"target_id"`
	TargetName       string            `json:"target_name"`
	TargetAvatarHash string            `json:"target_avatar_hash"`
	Name             string            `json:"name"`
	Success          bool              `json:"success"`
	ServerID         int               `json:"server_id"`
	ServerName       string            `json:"server_name"`
	Code             logparse.VoteCode `json:"code"`
	CreatedOn        time.Time         `json:"created_on"`
}
