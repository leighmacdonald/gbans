package maps

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/database"
)

type Repository struct {
	database.Database
}

func (r Repository) All(ctx context.Context) ([]Map, error) {
	var maps []Map
	rows, errRows := r.Query(ctx, `SELECT map_id, map_name, updated_on, created_on FROM map`)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var m Map
		if err := rows.Scan(&m.MapID, &m.MapName, &m.UpdatedOn, &m.CreatedOn); err != nil {
			return nil, database.Err(err)
		}
		maps = append(maps, m)
	}

	if rows.Err() != nil {
		return nil, database.Err(rows.Err())
	}

	return maps, nil
}

func NewRepository(db database.Database) Repository {
	return Repository{
		Database: db,
	}
}

func (r Repository) GetOrCreate(ctx context.Context, mapName string) (Map, error) {
	//nolint:unqueryvet
	const query = `
		WITH ins AS (
    		INSERT INTO map (map_id, map_name, updated_on, created_on) VALUES (DEFAULT, lower($1), now(),now())
    		ON CONFLICT (map_name) DO NOTHING RETURNING *
    	)
		SELECT * FROM ins
		UNION
		SELECT * FROM map
		WHERE map_name = lower($1);
		` //nolint:unqueryvet

	var mapDetail Map
	if errQuery := r.
		QueryRow(ctx, query, mapName).
		Scan(&mapDetail.MapID, &mapDetail.MapName, &mapDetail.UpdatedOn, &mapDetail.CreatedOn); errQuery != nil {
		return Map{}, database.Err(errQuery)
	}

	return mapDetail, nil
}

func (r Repository) GetByID(ctx context.Context, mapID int32) (Map, error) {
	const query = `SELECT map_id, map_name, updated_on, created_on FROM map WHERE map_id = $1`
	var mapDetail Map
	if errQuery := r.
		QueryRow(ctx, query, mapID).
		Scan(&mapDetail.MapID, &mapDetail.MapName, &mapDetail.UpdatedOn, &mapDetail.CreatedOn); errQuery != nil {
		return Map{}, database.Err(errQuery)
	}

	return mapDetail, nil
}
