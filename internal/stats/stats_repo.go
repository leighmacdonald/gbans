package stats

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
)

type Repository struct{ database.Database }

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) Insert(ctx context.Context) error {
	return nil
}

func (r Repository) AddPlayerStatsAlltime(ctx context.Context, stats demoparse.Stats) error {
	return nil
}
