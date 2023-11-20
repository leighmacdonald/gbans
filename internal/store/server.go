package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

type ServerPermission struct {
	SteamID         steamid.SID      `json:"steam_id"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	Flags           string           `json:"flags"`
}

func NewServer(shortName string, address string, port int) Server {
	return Server{
		ShortName:      shortName,
		Address:        address,
		Port:           port,
		RCON:           SecureRandomString(10),
		ReservedSlots:  0,
		Password:       SecureRandomString(10),
		IsEnabled:      true,
		TokenCreatedOn: time.Unix(0, 0),
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type Server struct {
	// Auto generated id
	ServerID int `db:"server_id" json:"server_id"`
	// ShortName is a short reference name for the server eg: us-1
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
	// Address is the ip of the server
	Address string `db:"address" json:"address"`
	// Port is the port of the server
	Port int `db:"port" json:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon" json:"rcon"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls (permanent refresh token)
	Password  string  `db:"password" json:"password"`
	IsEnabled bool    `json:"is_enabled"`
	Deleted   bool    `json:"deleted"`
	Region    string  `json:"region"`
	CC        string  `json:"cc"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	LogSecret int     `json:"log_secret"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn time.Time `db:"token_created_on" json:"token_created_on"`
	CreatedOn      time.Time `db:"created_on" json:"created_on"`
	UpdatedOn      time.Time `db:"updated_on" json:"updated_on"`
}

func (s Server) IP(ctx context.Context) (net.IP, error) {
	parsedIP := net.ParseIP(s.Address)
	if parsedIP != nil {
		// We already have an ip
		return parsedIP, nil
	}
	// TODO proper timeout for ctx
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.Address)
	if errResolve != nil || len(ips) == 0 {
		return nil, errors.Wrap(errResolve, "Could not resolve address")
	}

	return ips[0], nil
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

func (db *Store) GetServer(ctx context.Context, serverID int, server *Server) error {
	row, rowErr := db.QueryRowBuilder(ctx, db.sb.
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret").
		From(string(tableServer)).
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}))
	if rowErr != nil {
		return rowErr
	}

	if errScan := row.Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
		&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
		&server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret); errScan != nil {
		return Err(errScan)
	}

	return nil
}

func (db *Store) GetServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	rows, errRows := db.QueryBuilder(ctx, db.sb.
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": consts.PReserved}).
		OrderBy("permission_level desc"))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var perms []ServerPermission

	for rows.Next() {
		var (
			sid   int64
			perm  consts.Privilege
			flags string
		)

		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, Err(errScan)
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

		perms = append(perms, ServerPermission{
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

func (db *Store) GetServers(ctx context.Context, filter ServerQueryFilter) ([]Server, int64, error) {
	builder := db.sb.
		Select("s.server_id", "s.short_name", "s.name", "s.address", "s.port", "s.rcon", "s.password",
			"s.token_created_on", "s.created_on", "s.updated_on", "s.reserved_slots", "s.is_enabled", "s.region", "s.cc",
			"s.latitude", "s.longitude", "s.deleted", "s.log_secret").
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
			"latitude", "longitude", "deleted",
		},
	}, "short_name")

	builder = filter.applyLimitOffset(builder, 250).Where(constraints)

	rows, errQueryExec := db.QueryBuilder(ctx, builder)
	if errQueryExec != nil {
		return []Server{}, 0, Err(errQueryExec)
	}

	defer rows.Close()

	var servers []Server

	for rows.Next() {
		var server Server
		if errScan := rows.
			Scan(&server.ServerID, &server.ShortName, &server.Name, &server.Address, &server.Port, &server.RCON,
				&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
				&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
				&server.Deleted, &server.LogSecret); errScan != nil {
			return nil, 0, errors.Wrap(errScan, "Failed to scan server")
		}

		servers = append(servers, server)
	}

	if rows.Err() != nil {
		return nil, 0, Err(rows.Err())
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("count(s.server_id)").
		From("server s").
		Where(constraints))
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	return servers, count, nil
}

func (db *Store) GetServerByName(ctx context.Context, serverName string, server *Server, disabledOk bool, deletedOk bool) error {
	and := sq.And{sq.Eq{"short_name": serverName}}
	if !disabledOk {
		and = append(and, sq.Eq{"is_enabled": true})
	}

	if !deletedOk {
		and = append(and, sq.Eq{"deleted": false})
	}

	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("server_id", "short_name", "name", "address", "port", "rcon", "password",
			"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
			"latitude", "longitude", "deleted", "log_secret").
		From(string(tableServer)).
		Where(and))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(
		&server.ServerID,
		&server.ShortName,
		&server.Name,
		&server.Address,
		&server.Port,
		&server.RCON,
		&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
		&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
		&server.Deleted, &server.LogSecret))
}

// SaveServer updates or creates the server data in the database.
func (db *Store) SaveServer(ctx context.Context, server *Server) error {
	server.UpdatedOn = time.Now()
	if server.ServerID > 0 {
		return db.updateServer(ctx, server)
	}

	server.CreatedOn = time.Now()

	return db.insertServer(ctx, server)
}

func (db *Store) insertServer(ctx context.Context, server *Server) error {
	const query = `
		INSERT INTO server (
		    short_name, name, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, latitude, longitude, 
			deleted, log_secret) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING server_id;`

	err := db.QueryRow(ctx, query, server.ShortName, server.Name, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret).Scan(&server.ServerID)
	if err != nil {
		return Err(err)
	}

	return nil
}

func (db *Store) updateServer(ctx context.Context, server *Server) error {
	server.UpdatedOn = time.Now()

	return db.ExecUpdateBuilder(ctx, db.sb.
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
		Where(sq.Eq{"server_id": server.ServerID}))
}

func (db *Store) DropServer(ctx context.Context, serverID int) error {
	return db.ExecUpdateBuilder(ctx, db.sb.
		Update("server").
		Set("deleted", true).
		Where(sq.Eq{"server_id": serverID}))
}
