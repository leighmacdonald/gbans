package stats

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/database"
)

type Repository struct{ database.Database }

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) Insert(ctx context.Context) error {
	return nil
}
