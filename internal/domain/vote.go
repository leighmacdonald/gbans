package domain

import (
	"context"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"time"
)

type VoteQueryFilter struct {
	QueryFilter
	SourceID steamid.SteamID
	TargetID steamid.SteamID
	ServerID int
	MatchID  uuid.UUID
	Name     string
}

type VoteRepository interface {
	Query(ctx context.Context, filter VoteQueryFilter) ([]VoteResult, error)
	AddResult(ctx context.Context, voteResult VoteResult) error
}

type VoteUsecase interface {
	Query(ctx context.Context, filter VoteQueryFilter) ([]VoteResult, error)
	Start(ctx context.Context)
}

type VoteResult struct {
	ServerID  int
	MatchID   uuid.UUID
	SourceID  steamid.SteamID
	TargetID  steamid.SteamID
	Valid     int
	Name      string
	CreatedOn time.Time
}
