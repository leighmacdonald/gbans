package servers

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type serversRepository struct {
	db database.Database
}

func NewServersRepository(database database.Database) domain.ServersRepository {
	return &serversRepository{db: database}
}

func (r *serversRepository) GetServer(ctx context.Context, serverID int) (domain.Server, error) {
	var server domain.Server

	row, rowErr := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}))
	if rowErr != nil {
		return server, r.db.DBErr(rowErr)
	}

	var tokenTime *time.Time

	if errScan := row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
		&server.Password, &tokenTime, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
		&server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
		return server, r.db.DBErr(errScan)
	}

	if tokenTime != nil {
		server.TokenCreatedOn = *tokenTime
	}

	return server, nil
}

func (r *serversRepository) GetServerPermissions(ctx context.Context) ([]domain.ServerPermission, error) {
	rows, errRows := r.db.QueryBuilder(ctx, nil, r.db.
		Builder().
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": domain.PReserved}).
		OrderBy("permission_level desc"))
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var perms []domain.ServerPermission

	for rows.Next() {
		var (
			sid   steamid.SteamID
			perm  domain.Privilege
			flags string
		)

		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		switch perm {
		case domain.PReserved:
			flags = "a"
		case domain.PEditor:
			flags = "aj"
		case domain.PModerator:
			flags = "abcdegjk"
		case domain.PAdmin:
			flags = "z"
		}

		perms = append(perms, domain.ServerPermission{
			SteamID:         sid.Steam(false),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}

	return perms, nil
}

func (r *serversRepository) GetServers(ctx context.Context, filter domain.ServerQueryFilter) ([]domain.Server, int64, error) {
	builder := r.db.
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

	builder = filter.ApplySafeOrder(builder, map[string][]string{
		"s.": {
			"server_id", "short_name", "name", "address", "port",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "enable_stats",
		},
	}, "short_name")

	builder = filter.ApplyLimitOffset(builder, 250).Where(constraints)

	rows, errQueryExec := r.db.QueryBuilder(ctx, nil, builder)
	if errQueryExec != nil {
		return []domain.Server{}, 0, r.db.DBErr(errQueryExec)
	}

	defer rows.Close()

	var servers []domain.Server

	for rows.Next() {
		var (
			server    domain.Server
			tokenDate *time.Time
		)

		if errScan := rows.
			Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
				&server.Password, &tokenDate, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
				&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
				&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
			return nil, 0, errors.Join(errScan, domain.ErrScanResult)
		}

		if tokenDate != nil {
			server.TokenCreatedOn = *tokenDate
		}

		servers = append(servers, server)
	}

	if rows.Err() != nil {
		return nil, 0, r.db.DBErr(rows.Err())
	}

	count, errCount := r.db.GetCount(ctx, nil, r.db.
		Builder().
		Select("count(s.server_id)").
		From("server s").
		Where(constraints))
	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	return servers, count, nil
}

func (r *serversRepository) GetServerByName(ctx context.Context, serverName string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"short_name": serverName}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var tokenTime *time.Time

	if err := row.Scan(
		&server.ServerID,
		&server.ShortName,
		&server.Name,
		&server.Address,
		&server.Port,
		&server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
		&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret, &server.EnableStats); err != nil {
		return r.db.DBErr(err)
	}

	if tokenTime != nil {
		server.TokenCreatedOn = *tokenTime
	}

	return nil
}

func (r *serversRepository) GetServerByPassword(ctx context.Context, serverPassword string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"password": serverPassword}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var tokenTime *time.Time

	if err := row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port,
		&server.RCON, &server.Password, &tokenTime, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC, &server.Latitude,
		&server.Longitude, &server.Deleted, &server.LogSecret, &server.EnableStats); err != nil {
		return r.db.DBErr(err)
	}

	if tokenTime != nil {
		server.TokenCreatedOn = *tokenTime
	}

	return nil
}

// SaveServer updates or creates the server data in the database.
func (r *serversRepository) SaveServer(ctx context.Context, server *domain.Server) error {
	if server.ServerID > 0 {
		return r.updateServer(ctx, server)
	}

	return r.insertServer(ctx, server)
}

func (r *serversRepository) insertServer(ctx context.Context, server *domain.Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude, 
			deleted, log_secret, enable_stats) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING server_id;`

	err := r.db.QueryRow(ctx, nil, query, server.ShortName, server.Name, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret, &server.EnableStats).
		Scan(&server.ServerID)
	if err != nil {
		return r.db.DBErr(err)
	}

	return nil
}

func (r *serversRepository) updateServer(ctx context.Context, server *domain.Server) error {
	server.UpdatedOn = time.Now()

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.
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
