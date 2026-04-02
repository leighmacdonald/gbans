package maps

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/database"
)

type Repository struct {
	database.Database
}

func NewRepository(db database.Database) Repository {
	return Repository{
		Database: db,
	}
}

func (r Repository) GetOrCreate(ctx context.Context, mapName string) (Map, error) {
	const query = `
		WITH ins AS (
    		INSERT INTO map (map_id, map_name, updated_on, created_on) VALUES (DEFAULT, lower($1), now(),now())
    		ON CONFLICT (map_name) DO NOTHING RETURNING *
    	)
		SELECT * FROM ins
		UNION
		SELECT * FROM map
		WHERE map_name = lower($1);
		`

	var mapDetail Map
	if errQuery := r.
		QueryRow(ctx, query, mapName).
		Scan(&mapDetail.MapID, &mapDetail.MapName, &mapDetail.UpdatedOn, &mapDetail.CreatedOn); errQuery != nil {
		return Map{}, database.Err(errQuery)
	}

	return mapDetail, nil
}
