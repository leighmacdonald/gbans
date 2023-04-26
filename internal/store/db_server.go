package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
)

var columnsServer = []string{"server_id", "short_name", "name", "address", "port", "rcon", "password",
	"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
	"latitude", "longitude", "default_map", "deleted", "log_secret"}

func (database *pgStore) GetServer(ctx context.Context, serverID int, server *model.Server) error {
	query, args, errQuery := sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	if errRow := database.conn.QueryRow(ctx, query, args...).
		Scan(&server.ServerID, &server.ServerNameShort, &server.ServerNameLong, &server.Address, &server.Port, &server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
			&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
			&server.Latitude, &server.Longitude,
			&server.DefaultMap, &server.Deleted, &server.LogSecret); errRow != nil {
		return Err(errRow)
	}
	return nil
}

func (database *pgStore) GetServerPermissions(ctx context.Context) ([]model.ServerPermission, error) {
	query, args, errQuery := sb.
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": model.PReserved}).
		OrderBy("permission_level desc").
		ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	rows, errRows := database.Query(ctx, query, args...)
	if errRows != nil {
		return nil, Err(errRows)
	}
	defer rows.Close()
	var perms []model.ServerPermission
	for rows.Next() {
		var (
			sid  steamid.SID64
			perm model.Privilege
		)
		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, Err(errScan)
		}
		flags := ""
		switch perm {
		case model.PReserved:
			flags = "a"
		case model.PEditor:
			flags = "aj"
		case model.PModerator:
			flags = "abcdegjk"
		case model.PAdmin:
			flags = "z"
		}
		perms = append(perms, model.ServerPermission{
			SteamId:         steamid.SID64ToSID(sid),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}
	return perms, nil
}

func (database *pgStore) GetServers(ctx context.Context, includeDisabled bool) ([]model.Server, error) {
	var servers []model.Server
	queryBuilder := sb.Select(columnsServer...).From(string(tableServer))
	cond := sq.And{sq.Eq{"deleted": false}}
	if !includeDisabled {
		cond = append(cond, sq.Eq{"is_enabled": true})
	}
	queryBuilder = queryBuilder.Where(cond)
	query, args, errQuery := queryBuilder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	rows, errQueryExec := database.conn.Query(ctx, query, args...)
	if errQueryExec != nil {
		return []model.Server{}, errQueryExec
	}
	defer rows.Close()
	for rows.Next() {
		var server model.Server
		if errScan := rows.Scan(&server.ServerID, &server.ServerNameShort, &server.ServerNameLong, &server.Address, &server.Port, &server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
			&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
			&server.DefaultMap, &server.Deleted, &server.LogSecret); errScan != nil {
			return nil, errScan
		}
		servers = append(servers, server)
	}
	if rows.Err() != nil {
		return nil, Err(rows.Err())
	}
	return servers, nil
}

func (database *pgStore) GetServerByName(ctx context.Context, serverName string, server *model.Server) error {
	query, args, errQueryArgs := sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.And{sq.Eq{"short_name": serverName}, sq.Eq{"deleted": false}}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	return Err(database.conn.QueryRow(ctx, query, args...).
		Scan(
			&server.ServerID,
			&server.ServerNameShort,
			&server.ServerNameLong,
			&server.Address,
			&server.Port,
			&server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
			&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
			&server.DefaultMap, &server.Deleted, &server.LogSecret))
}

// SaveServer updates or creates the server data in the database
func (database *pgStore) SaveServer(ctx context.Context, server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return database.updateServer(ctx, server)
	}
	server.CreatedOn = config.Now()
	return database.insertServer(ctx, server)
}

func (database *pgStore) insertServer(ctx context.Context, server *model.Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude, 
			default_map, deleted, log_secret) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING server_id;`
	err := database.conn.QueryRow(ctx, query, server.ServerNameShort, server.ServerNameLong, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.DefaultMap, server.Deleted, &server.LogSecret).Scan(&server.ServerID)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) updateServer(ctx context.Context, server *model.Server) error {
	server.UpdatedOn = config.Now()
	query, args, errQueryArgs := sb.Update(string(tableServer)).
		Set("short_name", server.ServerNameShort).
		Set("name", server.ServerNameLong).
		Set("address", server.Address).
		Set("port", server.Port).
		Set("rcon", server.RCON).
		Set("token_created_on", server.TokenCreatedOn).
		Set("updated_on", server.UpdatedOn).
		Set("reserved_slots", server.ReservedSlots).
		Set("password", server.Password).
		Set("is_enabled", server.IsEnabled).
		Set("deleted", server.Deleted).
		Set("region", server.Region).
		Set("cc", server.CC).
		Set("latitude", server.Latitude).
		Set("longitude", server.Longitude).
		Set("default_map", server.DefaultMap).
		Set("log_secret", server.LogSecret).
		Where(sq.Eq{"server_id": server.ServerID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update server")
	}
	return nil
}

func (database *pgStore) DropServer(ctx context.Context, serverID int) error {
	const query = `UPDATE server set deleted = true WHERE server_id = $1`
	if _, errExec := database.conn.Exec(ctx, query, serverID); errExec != nil {
		return errExec
	}
	return nil
}
