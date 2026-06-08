package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
)

const (
	MinPlayers = 4
	MinDuraion = 300
)

var ErrInvalidState = errors.New("invalid demo state")
var ErrInvalidBucket = errors.New("invalid stat bucket")

type Stats struct {
	repo Repository
	maps maps.Maps
}

func New(repo Repository, maps maps.Maps) Stats {
	return Stats{repo: repo, maps: maps}
}

type Bucket struct {
	BucketID   int32
	BucketName string
}

func (s Stats) Bucket(ctx context.Context, bucketID int32) (*Bucket, error) {
	if bucketID <= 0 {
		return nil, ErrInvalidBucket
	}

	return s.repo.GetBucket(ctx, bucketID)
}

func (s Stats) Import(ctx context.Context, serverID int32, demoID int32, demo *demoparse.Demo, timeStart time.Time) (*uuid.UUID, error) {
	if demo.DemoType != demoparse.HL2Demo {
		return nil, fmt.Errorf("%w: invalid demo type", ErrInvalidState)
	}

	if demo.Server == "" {
		return nil, fmt.Errorf("%w: invalid server name", ErrInvalidState)
	}

	if demo.Filename == "" {
		return nil, fmt.Errorf("%w: invalid file name", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if len(demo.Rounds) == 0 {
		return nil, fmt.Errorf("%w not enough rounds", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if demo.Map == "" {
		return nil, fmt.Errorf("%w: empty map invalid", ErrInvalidState)
	}

	mapInfo, errMap := s.maps.Get(ctx, demo.Map)
	if errMap != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidState, errMap)
	}

	matchID, errMatch := s.repo.CreateMatch(ctx, serverID, demoID, demo, timeStart, mapInfo, nil)
	if errMatch != nil {
		return nil, errMatch
	}

	return &matchID, nil
}
