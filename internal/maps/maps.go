package maps

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidMap = errors.New("invalid map")

type Map struct {
	MapID     int32
	MapName   string
	CreatedOn time.Time
	UpdatedOn time.Time
}

type Maps struct {
	repo Repository
}

func New(repo Repository) Maps {
	return Maps{
		repo: repo,
	}
}

func (m Maps) Get(ctx context.Context, name string) (Map, error) {
	if name == "" {
		return Map{}, fmt.Errorf("%w: empty map name", ErrInvalidMap)
	}

	return m.repo.GetOrCreate(ctx, name)
}

func (m Maps) GetByID(ctx context.Context, mapID int32) (Map, error) {
	if mapID == 0 {
		return Map{}, fmt.Errorf("%w: zero map id", ErrInvalidMap)
	}

	return m.repo.GetByID(ctx, mapID)
}

func (m Maps) All(ctx context.Context) ([]Map, error) {
	return m.repo.All(ctx)
}
