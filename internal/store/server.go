package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

type ServerPermission struct {
	SteamID         steamid.SID      `json:"steam_id"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	Flags           string           `json:"flags"`
}

func NewServer(name string, address string, port int) Server {
	return Server{
		ServerNameShort: name,
		Address:         address,
		Port:            port,
		RCON:            "",
		ReservedSlots:   0,
		Password:        "",
		IsEnabled:       true,
		TokenCreatedOn:  time.Unix(0, 0),
		CreatedOn:       config.Now(),
		UpdatedOn:       config.Now(),
	}
}

type Server struct {
	// Auto generated id
	ServerID int `db:"server_id" json:"server_id"`
	// ServerNameShort is a short reference name for the server eg: us-1
	ServerNameShort string `db:"short_name" json:"server_name"`
	ServerNameLong  string `db:"server_name_long" json:"server_name_long"`
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

var columnsServer = []string{
	"server_id", "short_name", "name", "address", "port", "rcon", "password",
	"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
	"latitude", "longitude", "deleted", "log_secret",
}

func (db *Store) GetServer(ctx context.Context, serverID int, server *Server) error {
	query, args, errQuery := db.sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.And{sq.Eq{"server_id": serverID}, sq.Eq{"deleted": false}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	if errRow := db.QueryRow(ctx, query, args...).
		Scan(&server.ServerID, &server.ServerNameShort, &server.ServerNameLong, &server.Address, &server.Port, &server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn,
			&server.ReservedSlots, &server.IsEnabled, &server.Region, &server.CC,
			&server.Latitude, &server.Longitude,
			&server.Deleted, &server.LogSecret); errRow != nil {
		return Err(errRow)
	}
	return nil
}

func (db *Store) GetServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	query, args, errQuery := db.sb.
		Select("steam_id", "permission_level").From("person").
		Where(sq.GtOrEq{"permission_level": consts.PReserved}).
		OrderBy("permission_level desc").
		ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		return nil, Err(errRows)
	}
	defer rows.Close()
	var perms []ServerPermission
	for rows.Next() {
		var (
			sid  steamid.SID64
			perm consts.Privilege
		)
		if errScan := rows.Scan(&sid, &perm); errScan != nil {
			return nil, Err(errScan)
		}
		flags := ""
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
			SteamID:         steamid.SID64ToSID(sid),
			PermissionLevel: perm,
			Flags:           flags,
		})
	}
	return perms, nil
}

func (db *Store) GetServers(ctx context.Context, includeDisabled bool) ([]Server, error) {
	var servers []Server
	queryBuilder := db.sb.Select(columnsServer...).From(string(tableServer))
	cond := sq.And{sq.Eq{"deleted": false}}
	if !includeDisabled {
		cond = append(cond, sq.Eq{"is_enabled": true})
	}
	queryBuilder = queryBuilder.Where(cond)
	query, args, errQuery := queryBuilder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	rows, errQueryExec := db.Query(ctx, query, args...)
	if errQueryExec != nil {
		return []Server{}, errQueryExec
	}
	defer rows.Close()
	for rows.Next() {
		var server Server
		if errScan := rows.Scan(&server.ServerID, &server.ServerNameShort, &server.ServerNameLong, &server.Address, &server.Port, &server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
			&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
			&server.Deleted, &server.LogSecret); errScan != nil {
			return nil, errScan
		}
		servers = append(servers, server)
	}
	if rows.Err() != nil {
		return nil, Err(rows.Err())
	}
	return servers, nil
}

func (db *Store) GetServerByName(ctx context.Context, serverName string, server *Server) error {
	query, args, errQueryArgs := db.sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.And{sq.Eq{"short_name": serverName}, sq.Eq{"deleted": false}}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	return Err(db.QueryRow(ctx, query, args...).
		Scan(
			&server.ServerID,
			&server.ServerNameShort,
			&server.ServerNameLong,
			&server.Address,
			&server.Port,
			&server.RCON,
			&server.Password, &server.TokenCreatedOn, &server.CreatedOn, &server.UpdatedOn, &server.ReservedSlots,
			&server.IsEnabled, &server.Region, &server.CC, &server.Latitude, &server.Longitude,
			&server.Deleted, &server.LogSecret))
}

// SaveServer updates or creates the server data in the database.
func (db *Store) SaveServer(ctx context.Context, server *Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return db.updateServer(ctx, server)
	}
	server.CreatedOn = config.Now()
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
	err := db.QueryRow(ctx, query, server.ServerNameShort, server.ServerNameLong, server.Address, server.Port,
		server.RCON, server.TokenCreatedOn, server.ReservedSlots, server.CreatedOn, server.UpdatedOn,
		server.Password, server.IsEnabled, server.Region, server.CC,
		server.Latitude, server.Longitude, server.Deleted, &server.LogSecret).Scan(&server.ServerID)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (db *Store) updateServer(ctx context.Context, server *Server) error {
	server.UpdatedOn = config.Now()
	query, args, errQueryArgs := db.sb.Update(string(tableServer)).
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
		Set("log_secret", server.LogSecret).
		Where(sq.Eq{"server_id": server.ServerID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errExec := db.exec(ctx, query, args...); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update server")
	}
	return nil
}

func (db *Store) DropServer(ctx context.Context, serverID int) error {
	const query = `UPDATE server set deleted = true WHERE server_id = $1`
	if errExec := db.exec(ctx, query, serverID); errExec != nil {
		return errExec
	}
	return nil
}
