package servers

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	db database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{db: database}
}

// todo move to srcds.
func (r *Repository) GetServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("steam_id", "permission_level").
		From("person").
		Where(sq.GtOrEq{"permission_level": permission.Reserved}).
		OrderBy("permission_level desc"))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var perms []ServerPermission

	for rows.Next() {
		var (
			sid   steamid.SteamID
			perm  permission.Privilege
			flags string
		)

		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		switch perm {
		case permission.Reserved:
			flags = "a"
		case permission.Editor:
			flags = "aj"
		case permission.Moderator:
			flags = "abcdegjk"
		case permission.Admin:
			flags = "z"
		}

		perms = append(perms, ServerPermission{
			SteamID:         sid.Steam(false),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}

	return perms, nil
}

func (r *Repository) Query(ctx context.Context, filter Query) ([]Server, error) {
	builder := r.db.
		Builder().
		Select("s.server_id", "s.short_name", "s.name", "s.address", "s.port", "s.rcon", "s.password",
			"s.token_created_on", "s.created_on", "s.updated_on", "s.reserved_slots", "s.is_enabled", "s.region", "s.cc",
			"s.latitude", "s.longitude", "s.deleted", "s.log_secret", "s.enable_stats", "s.address_internal", "s.sdr_enabled").
		From("server s")

	var constraints sq.And

	if filter.ServerID > 0 {
		constraints = append(constraints, sq.Eq{"s.server_id": filter.ServerID})
	}

	if !filter.IncludeDisabled {
		constraints = append(constraints, sq.Eq{"s.is_enabled": true})
	}

	if filter.SDROnly {
		constraints = append(constraints, sq.Eq{"s.sdr_enabled": true})
	}

	if !filter.IncludeDeleted {
		constraints = append(constraints, sq.Eq{"s.deleted": false})
	}

	if filter.ShortName != "" {
		constraints = append(constraints, sq.Eq{"s.short_name": filter.ShortName})
	}

	if filter.Password != "" {
		constraints = append(constraints, sq.Eq{"s.password": filter.Password})
	}

	rows, errQueryExec := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errQueryExec != nil {
		return []Server{}, database.DBErr(errQueryExec)
	}

	defer rows.Close()

	//goland:noinspection GoPreferNilSlice
	servers := []Server{}

	for rows.Next() {
		var (
			server    Server
			tokenDate *time.Time
		)

		if errScan := rows.
			Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
				&server.Password, &tokenDate, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
				&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
				&server.Deleted, &server.LogSecret, &server.EnableStats, &server.AddressInternal, &server.SDREnabled); errScan != nil {
			return nil, errors.Join(errScan, database.ErrScanResult)
		}

		if tokenDate != nil {
			server.TokenCreatedOn = *tokenDate
		}

		servers = append(servers, server)
	}

	if rows.Err() != nil {
		return nil, database.DBErr(rows.Err())
	}

	return servers, nil
}

// SaveServer updates or creates the server data in the database.
func (r *Repository) Save(ctx context.Context, server *Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on,
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude,
			deleted, log_secret, enable_stats, address_internal, sdr_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (short_name) DO UPDATE SET
			name = $2, address = $3, port = $4, rcon = $5, token_created_on = $6, reserved_slots = $7,
			updated_on = $8, password = $10, is_enabled = $11, region = $12, cc = $13, latitude = $14, longitude = $15,
      		deleted = $16, log_secret = $17, enable_stats = $18, address_internal = $19, sdr_enabled = $20
		RETURNING server_id;`

	err := r.db.QueryRow(ctx, query, server.ShortName, server.Name, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret, &server.EnableStats, &server.AddressInternal, &server.SDREnabled).
		Scan(&server.ServerID)
	if err != nil {
		return database.DBErr(err)
	}

	return nil
}
