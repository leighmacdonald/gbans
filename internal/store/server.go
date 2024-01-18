package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

func GetServer(ctx context.Context, database Store, serverID int, server *model.Server) error {
	row, rowErr := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}))
	if rowErr != nil {
		return DBErr(rowErr)
	}

	if errScan := row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
		&server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
		return DBErr(errScan)
	}

	return nil
}

func GetServerPermissions(ctx context.Context, database Store) ([]model.ServerPermission, error) {
	rows, errRows := database.QueryBuilder(ctx, database.
		Builder().
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": consts.PReserved}).
		OrderBy("permission_level desc"))
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	var perms []model.ServerPermission

	for rows.Next() {
		var (
			sid   int64
			perm  consts.Privilege
			flags string
		)

		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, DBErr(errScan)
		}

		switch perm {
		case consts.PReserved:
			flags = "a"
		case consts.PEditor:
			flags = "aj"
		case consts.PModerator:
			flags = "abcdegjk"
		case consts.PAdmin:
			flags = "z"
		}

		perms = append(perms, model.ServerPermission{
			SteamID:         steamid.SID64ToSID(steamid.New(sid)),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}

	return perms, nil
}

type ServerQueryFilter struct {
	QueryFilter
	IncludeDisabled bool `json:"include_disabled"`
}

func GetServers(ctx context.Context, database Store, filter ServerQueryFilter) ([]model.Server, int64, error) {
	builder := database.
		Builder().
		Select("s.server_id", "s.short_name", "s.name", "s.address", "s.port", "s.rcon", "s.password",
			"s.token_created_on", "s.created_on", "s.updated_on", "s.reserved_slots", "s.is_enabled", "s.region", "s.cc",
			"s.latitude", "s.longitude", "s.deleted", "s.log_secret", "s.enable_stats").
		From("server s")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"s.deleted": false})
	}

	if !filter.IncludeDisabled {
		constraints = append(constraints, sq.Eq{"s.is_enabled": true})
	}

	builder = filter.applySafeOrder(builder, map[string][]string{
		"s.": {
			"server_id", "short_name", "name", "address", "port",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "enable_stats",
		},
	}, "short_name")

	builder = filter.applyLimitOffset(builder, 250).Where(constraints)

	rows, errQueryExec := database.QueryBuilder(ctx, builder)
	if errQueryExec != nil {
		return []model.Server{}, 0, DBErr(errQueryExec)
	}

	defer rows.Close()

	var servers []model.Server

	for rows.Next() {
		var server model.Server
		if errScan := rows.
			Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
				&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
				&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
				&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
			return nil, 0, errors.Wrap(errScan, "Failed to scan server")
		}

		servers = append(servers, server)
	}

	if rows.Err() != nil {
		return nil, 0, DBErr(rows.Err())
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("count(s.server_id)").
		From("server s").
		Where(constraints))
	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	return servers, count, nil
}

func GetServerByName(ctx context.Context, database Store, serverName string, server *model.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"short_name": serverName}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(
		&server.ServerID,
		&server.ShortName,
		&server.Name,
		&server.Address,
		&server.Port,
		&server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
		&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret, &server.EnableStats))
}

func GetServerByPassword(ctx context.Context, database Store, serverPassword string, server *model.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"password": serverPassword}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port,
		&server.RCON, &server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC, &server.Latitude,
		&server.Longitude, &server.Deleted, &server.LogSecret, &server.EnableStats))
}

// SaveServer updates or creates the server data in the database.
func SaveServer(ctx context.Context, database Store, server *model.Server) error {
	server.UpdatedOn = time.Now()
	if server.ServerID > 0 {
		return updateServer(ctx, database, server)
	}

	server.CreatedOn = time.Now()

	return insertServer(ctx, database, server)
}

func insertServer(ctx context.Context, database Store, server *model.Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude, 
			deleted, log_secret, enable_stats) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING server_id;`

	err := database.QueryRow(ctx, query, server.ShortName, server.Name, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret, &server.EnableStats).
		Scan(&server.ServerID)
	if err != nil {
		return DBErr(err)
	}

	return nil
}

func updateServer(ctx context.Context, database Store, server *model.Server) error {
	server.UpdatedOn = time.Now()

	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("server").
		Set("short_name", server.ShortName).
		Set("name", server.Name).
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
		Set("log_secret", server.LogSecret).
		Set("enable_stats", server.EnableStats).
		Where(sq.Eq{"server_id": server.ServerID})))
}

func DropServer(ctx context.Context, database Store, serverID int) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("server").
		Set("deleted", true).
		Where(sq.Eq{"server_id": serverID})))
}
