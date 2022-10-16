package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
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

func (database *pgStore) FindLogEvents(ctx context.Context, opts model.LogQueryOpts) ([]model.ServerEvent, error) {
	queryBuilder := sb.Select(
		`l.log_id`,
		`s.server_id`,
		`l.event_type`,
		`l.created_on`,
		`s.short_name`,
		`COALESCE(source.steam_id, 0)`,
		`COALESCE(source.personaname, '')`,
		`COALESCE(source.avatarfull, '')`,
		`COALESCE(source.avatar, '')`,
		`COALESCE(target.steam_id, 0)`,
		`COALESCE(target.personaname, '')`,
		`COALESCE(target.avatarfull, '')`,
		`COALESCE(target.avatar, '')`,
		`l.weapon`,
		`l.damage`,
		`l.healing`,
		"COALESCE(ST_X(l.attacker_position::geometry), 0)",
		"COALESCE(ST_Y(l.attacker_position::geometry), 0)",
		"COALESCE(ST_Z(l.attacker_position::geometry), 0)",
		"COALESCE(ST_X(l.victim_position::geometry), 0)",
		"COALESCE(ST_Y(l.victim_position::geometry), 0)",
		"COALESCE(ST_Z(l.victim_position::geometry), 0)",
		"COALESCE(ST_X(l.assister_position::geometry), 0)",
		"COALESCE(ST_Y(l.assister_position::geometry), 0)",
		"COALESCE(ST_Z(l.assister_position::geometry), 0)",
		`l.item`,
		`l.player_class`,
		`l.player_team`,
		`l.meta_data`,
	).
		From("server_log l").
		LeftJoin(`server s on s.server_id = l.server_id`).
		LeftJoin(`person source on source.steam_id = l.source_id`).
		LeftJoin(`person target on target.steam_id = l.target_id`)

	if opts.Network != "" {
		_, network, errParseCIDR := net.ParseCIDR(opts.Network)
		if errParseCIDR != nil {
			return nil, Err(errParseCIDR)
		}
		idsByNet, errIdByNet := database.GetSteamIDsAtIP(ctx, network)
		if errIdByNet != nil {
			return nil, Err(errIdByNet)
		}
		queryBuilder = queryBuilder.Where(sq.Eq{"l.source_id": idsByNet})
	}
	sourceSid64, errSourceSid64 := steamid.StringToSID64(opts.SourceID)
	if opts.SourceID != "" && errSourceSid64 == nil && sourceSid64.Valid() {
		queryBuilder = queryBuilder.Where(sq.Eq{"l.source_id": sourceSid64.Int64()})
	}
	targetSid64, errTargetSid64 := steamid.StringToSID64(opts.TargetID)
	if opts.TargetID != "" && errTargetSid64 == nil && targetSid64.Valid() {
		queryBuilder = queryBuilder.Where(sq.Eq{"l.target_id": targetSid64.Int64()})
	}
	if len(opts.Servers) > 0 {
		queryBuilder = queryBuilder.Where(sq.Eq{"l.server_id": opts.Servers})
	}
	if len(opts.LogTypes) > 0 {
		queryBuilder = queryBuilder.Where(sq.Eq{"l.event_type": opts.LogTypes})
	}

	if opts.SentBefore != nil {
		queryBuilder = queryBuilder.Where(sq.Lt{"l.created_on": opts.SentBefore})
	}
	if opts.SentAfter != nil {
		queryBuilder = queryBuilder.Where(sq.Gt{"l.created_on": opts.SentAfter})
	}
	if opts.OrderDesc {
		queryBuilder = queryBuilder.OrderBy("l.created_on DESC")
	} else {
		queryBuilder = queryBuilder.OrderBy("l.created_on ASC")
	}
	if opts.Limit > 0 {
		queryBuilder = queryBuilder.Limit(opts.Limit)
	}
	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errQueryArgs
	}
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var events []model.ServerEvent
	for rows.Next() {
		event := model.NewServerEvent()
		if errScan := rows.Scan(
			&event.LogID, &event.Server.ServerID, &event.EventType, &event.CreatedOn,
			&event.Server.ServerNameShort,
			&event.Source.SteamID, &event.Source.PersonaName, &event.Source.AvatarFull, &event.Source.Avatar,
			&event.Target.SteamID, &event.Target.PersonaName, &event.Target.AvatarFull, &event.Target.Avatar,
			&event.Weapon, &event.Damage, &event.Healing,
			&event.AttackerPOS.X, &event.AttackerPOS.Y, &event.AttackerPOS.Z,
			&event.VictimPOS.X, &event.VictimPOS.Y, &event.VictimPOS.Z,
			&event.AssisterPOS.X, &event.AssisterPOS.Y, &event.AssisterPOS.Z,
			&event.Item, &event.PlayerClass, &event.Team, &event.MetaData); errScan != nil {
			return nil, Err(errScan)
		}
		events = append(events, event)
	}
	return events, nil
}

// BatchInsertServerLogs save server log events to the database using a
func (database *pgStore) BatchInsertServerLogs(ctx context.Context, serverEvents []model.ServerEvent) error {
	const (
		stmtName = "insert-log"
		query    = `INSERT INTO server_log (
		    server_id, event_type, source_id, target_id, created_on, weapon, damage, 
		    item, player_class, attacker_position, victim_position, assister_position,
            player_team, healing, meta_data
		) VALUES (
		    $1, $2, $3, $4, $5, $6, $7, $8, $9, 
		    CASE WHEN $10 != 0 AND $11 != 0 AND $12 != 0 THEN
		    	ST_SetSRID(ST_MakePoint($10, $11, $12), 4326)
		    END,
		    CASE WHEN $13 != 0 AND $14 != 0 AND $15 != 0 THEN
		    	ST_SetSRID(ST_MakePoint($13, $14, $13), 4326)
			END,
		    CASE WHEN $16 != 0 AND $17 != 0 AND $18 != 0 THEN
		          ST_SetSRID(ST_MakePoint($16, $17, $18), 4326)
			END, $19, $20, $21)`
	)
	t0 := config.Now()
	tx, errBeginTx := database.conn.Begin(ctx)
	if errBeginTx != nil {
		return errors.Wrapf(errBeginTx, "Failed to prepare logWriter query: %v", errBeginTx)
	}
	_, errPrepare := tx.Prepare(ctx, stmtName, query)
	if errPrepare != nil {
		return errors.Wrapf(errPrepare, "Failed to prepare logWriter query: %v", errPrepare)
	}
	var errExec error
	for _, serverEvent := range serverEvents {
		if serverEvent.Server.ServerID == 0 || serverEvent.Server.ServerID <= 0 {
			continue
		}
		source := steamid.SID64(0)
		target := steamid.SID64(0)
		if !serverEvent.Source.SteamID.Valid() {
			source = serverEvent.Source.SteamID
		}
		if !serverEvent.Target.SteamID.Valid() {
			target = serverEvent.Target.SteamID
		}

		if _, errExec = tx.Exec(ctx, stmtName, serverEvent.Server.ServerID, serverEvent.EventType,
			source.Int64(), target.Int64(), serverEvent.CreatedOn, serverEvent.Weapon, serverEvent.Damage,
			serverEvent.Item, serverEvent.PlayerClass,
			serverEvent.AttackerPOS.Y, serverEvent.AttackerPOS.X, serverEvent.AttackerPOS.Z,
			serverEvent.VictimPOS.Y, serverEvent.VictimPOS.X, serverEvent.VictimPOS.Z,
			serverEvent.AssisterPOS.Y, serverEvent.AssisterPOS.X, serverEvent.AssisterPOS.Z,
			serverEvent.Team, serverEvent.Healing, serverEvent.MetaData); errExec != nil {
			errExec = errors.Wrapf(errExec, "Failed to write log entries")
			break
		}
	}
	if errExec != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			return errors.Wrapf(errRollback, "BatchInsertServerLogs rollback failed")
		}
		return errors.Wrapf(errExec, "Failed to commit log entries")
	}
	if errCommit := tx.Commit(ctx); errCommit != nil {
		return errors.Wrapf(errCommit, "Failed to commit log entries")
	}
	log.WithFields(log.Fields{"count": len(serverEvents), "duration": time.Since(t0).String()}).
		Debug("Wrote event serverEvents successfully")
	return nil
}
