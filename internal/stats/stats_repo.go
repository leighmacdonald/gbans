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

func (r Repository) Insert(_ context.Context) error {
	return nil
}

func (r Repository) AddPlayerStatsAlltime(_ context.Context, _ demoparse.Stats) error {
	return nil
}
