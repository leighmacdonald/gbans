package maps

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidMap = errors.New("invalid map")

type Map struct {
	MapID     int       `json:"map_id"`
	MapName   string    `json:"map_name"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
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
