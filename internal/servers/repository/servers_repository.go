package repository

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type serversRepository struct {
	store.Database
}

func NewServersRepository(database store.Database) domain.ServersRepository {
	return &serversRepository{Database: database}
}

func (s *serversRepository) GetServer(ctx context.Context, serverID int, server *domain.Server) error {
	row, rowErr := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}))
	if rowErr != nil {
		return errs.DBErr(rowErr)
	}

	if errScan := row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
		&server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s *serversRepository) GetServerPermissions(ctx context.Context) ([]domain.ServerPermission, error) {
	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": domain.PReserved}).
		OrderBy("permission_level desc"))
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	var perms []domain.ServerPermission

	for rows.Next() {
		var (
			sid   int64
			perm  domain.Privilege
			flags string
		)

		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, errs.DBErr(errScan)
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
			SteamID:         steamid.SID64ToSID(steamid.New(sid)),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}

	return perms, nil
}

func (s *serversRepository) GetServers(ctx context.Context, filter domain.ServerQueryFilter) ([]domain.Server, int64, error) {
	builder := s.
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

	rows, errQueryExec := s.QueryBuilder(ctx, builder)
	if errQueryExec != nil {
		return []domain.Server{}, 0, errs.DBErr(errQueryExec)
	}

	defer rows.Close()

	var servers []domain.Server

	for rows.Next() {
		var server domain.Server
		if errScan := rows.
			Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
				&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
				&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
				&server.Deleted, &server.LogSecret, &server.EnableStats); errScan != nil {
			return nil, 0, errors.Join(errScan, domain.ErrScanResult)
		}

		servers = append(servers, server)
	}

	if rows.Err() != nil {
		return nil, 0, errs.DBErr(rows.Err())
	}

	count, errCount := s.GetCount(ctx, s, s.
		Builder().
		Select("count(s.server_id)").
		From("server s").
		Where(constraints))
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	return servers, count, nil
}

func (s *serversRepository) GetServerByName(ctx context.Context, serverName string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"short_name": serverName}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	return errs.DBErr(row.Scan(
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

func (s *serversRepository) GetServerByPassword(ctx context.Context, serverPassword string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"password": serverPassword}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret", "enable_stats").
		From("server").
		Where(and))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	return errs.DBErr(row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port,
		&server.RCON, &server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC, &server.Latitude,
		&server.Longitude, &server.Deleted, &server.LogSecret, &server.EnableStats))
}

// SaveServer updates or creates the server data in the database.
func (s *serversRepository) SaveServer(ctx context.Context, server *domain.Server) error {
	server.UpdatedOn = time.Now()
	if server.ServerID > 0 {
		return s.updateServer(ctx, server)
	}

	server.CreatedOn = time.Now()

	return s.insertServer(ctx, server)
}

func (s *serversRepository) insertServer(ctx context.Context, server *domain.Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude, 
			deleted, log_secret, enable_stats) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING server_id;`

	err := s.QueryRow(ctx, query, server.ShortName, server.Name, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret, &server.EnableStats).
		Scan(&server.ServerID)
	if err != nil {
		return errs.DBErr(err)
	}

	return nil
}

func (s *serversRepository) updateServer(ctx context.Context, server *domain.Server) error {
	server.UpdatedOn = time.Now()

	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
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

func (s *serversRepository) DropServer(ctx context.Context, serverID int) error {
	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("server").
		Set("deleted", true).
		Where(sq.Eq{"server_id": serverID})))
}