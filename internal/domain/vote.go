package domain

import (
	"context"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type VoteRepository interface {
}

type VoteUsecase interface {
	Start(ctx context.Context)
}

type VoteResult struct {
	ServerID int
	MatchID  uuid.UUID
	SourceID steamid.SteamID
	TargetID steamid.SteamID
	Valid    int
	Name     string
	Proxy    int
}
