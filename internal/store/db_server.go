package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var columnsServer = []string{"server_id", "short_name", "token", "address", "port", "rcon", "password",
	"token_created_on", "created_on", "updated_on", "reserved_slots", "is_enabled", "region", "cc",
	"ST_X(location::geometry)", "ST_Y(location::geometry)", "default_map"}

func (db *pgStore) GetServer(ctx context.Context, serverID int64, s *model.Server) error {
	q, a, e := sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.Eq{"server_id": serverID}).
		ToSql()
	if e != nil {
		return e
	}
	if err := db.c.QueryRow(ctx, q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port, &s.RCON,
			&s.Password, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn,
			&s.ReservedSlots, &s.IsEnabled, &s.Region, &s.CC, &s.Location.Longitude, &s.Location.Latitude,
			&s.DefaultMap); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *pgStore) GetServers(ctx context.Context, includeDisabled bool) ([]model.Server, error) {
	var servers []model.Server
	var qb sq.SelectBuilder
	qb = sq.Select(columnsServer...).From(string(tableServer))
	if !includeDisabled {
		sb = sb.Where(sq.Eq{"is_enabled": true})
	}
	q, a, e := qb.ToSql()
	if e != nil {
		return nil, dbErr(e)
	}
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return []model.Server{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.Server
		if err2 := rows.Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port, &s.RCON,
			&s.Password, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots,
			&s.IsEnabled, &s.Region, &s.CC, &s.Location.Longitude, &s.Location.Latitude,
			&s.DefaultMap); err2 != nil {
			return nil, err2
		}
		servers = append(servers, s)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return servers, nil
}

func (db *pgStore) GetServerByName(ctx context.Context, serverName string, s *model.Server) error {
	q, a, e := sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.Eq{"short_name": serverName}).
		ToSql()
	if e != nil {
		return e
	}
	if err := db.c.QueryRow(ctx, q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port, &s.RCON,
			&s.Password, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots,
			&s.IsEnabled, &s.Region, &s.CC, &s.Location.Longitude, &s.Location.Latitude,
			&s.DefaultMap); err != nil {
		return err
	}
	return nil
}

// SaveServer updates or creates the server data in the database
func (db *pgStore) SaveServer(ctx context.Context, server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return db.updateServer(ctx, server)
	}
	server.CreatedOn = config.Now()
	return db.insertServer(ctx, server)
}

func (db *pgStore) insertServer(ctx context.Context, s *model.Server) error {
	const q = `
		INSERT INTO server (
		    short_name, token, address, port, rcon, token_created_on, 
		    reserved_slots, created_on, updated_on, password, is_enabled, region, cc, location, default_map) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING server_id;`
	err := db.c.QueryRow(ctx, q, s.ServerName, s.Token, s.Address, s.Port, s.RCON, s.TokenCreatedOn,
		s.ReservedSlots, s.CreatedOn, s.UpdatedOn, s.Password, s.IsEnabled, s.Region, s.CC,
		s.Location.String(), s.DefaultMap).Scan(&s.ServerID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *pgStore) updateServer(ctx context.Context, s *model.Server) error {
	s.UpdatedOn = config.Now()
	q, a, e := sb.Update(string(tableServer)).
		Set("short_name", s.ServerName).
		Set("token", s.Token).
		Set("address", s.Address).
		Set("port", s.Port).
		Set("rcon", s.RCON).
		Set("token_created_on", s.TokenCreatedOn).
		Set("updated_on", s.UpdatedOn).
		Set("reserved_slots", s.ReservedSlots).
		Set("password", s.Password).
		Set("is_enabled", s.IsEnabled).
		Set("region", s.Region).
		Set("cc", s.CC).
		Set("location", s.Location.String()).
		Set("default_map", s.DefaultMap).
		Where(sq.Eq{"server_id": s.ServerID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update s")
	}
	return nil
}

func (db *pgStore) DropServer(ctx context.Context, serverID int64) error {
	q, a, e := sb.Delete(string(tableServer)).Where(sq.Eq{"server_id": serverID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return err
	}
	return nil
}

func (db *pgStore) FindLogEvents(ctx context.Context, opts model.LogQueryOpts) ([]model.ServerEvent, error) {
	b := sb.Select(
		`l.log_id`,
		`l.event_type`,
		`l.created_on`,
		`s.server_id`,
		`s.short_name`,
		`COALESCE(source.steam_id, 0)`,
		`COALESCE(source.personaname, '')`,
		`COALESCE(source.avatarfull, '')`,
		`COALESCE(source.avatar, '')`,
		`COALESCE(target.steam_id, 0)`,
		`COALESCE(target.personaname, '')`,
		`COALESCE(target.avatarfull, '')`,
		`COALESCE(target.avatar, '')`,
	).
		From("server_log l").
		LeftJoin(`server  s on s.server_id = l.server_id`).
		LeftJoin(`person source on source.steam_id = l.source_id`).
		LeftJoin(`person target on target.steam_id = l.target_id`)

	s1, e1 := steamid.StringToSID64(opts.SourceID)
	if opts.SourceID != "" && e1 == nil && s1.Valid() {
		b = b.Where(sq.Eq{"l.source_id": s1.Int64()})
	}
	t1, e2 := steamid.StringToSID64(opts.TargetID)
	if opts.TargetID != "" && e2 == nil && t1.Valid() {
		b = b.Where(sq.Eq{"l.target_id": t1.Int64()})
	}
	if len(opts.Servers) > 0 {
		b = b.Where(sq.Eq{"l.server_id": opts.Servers})
	}
	if len(opts.LogTypes) > 0 {
		b = b.Where(sq.Eq{"l.event_type": opts.LogTypes})
	}
	if opts.OrderDesc {
		b = b.OrderBy("l.created_on DESC")
	} else {
		b = b.OrderBy("l.created_on ASC")
	}
	if opts.Limit > 0 {
		b = b.Limit(opts.Limit)
	}
	q, a, err := b.ToSql()
	log.Debugf(q)
	if err != nil {
		return nil, err
	}
	rows, errQ := db.c.Query(ctx, q, a...)
	if errQ != nil {
		return nil, dbErr(errQ)
	}
	defer rows.Close()
	var events []model.ServerEvent
	for rows.Next() {
		e := model.ServerEvent{
			Server: &model.Server{},
			Source: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
			Target: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
		}
		if err2 := rows.Scan(
			&e.LogID, &e.EventType, &e.CreatedOn,
			&e.Server.ServerID, &e.Server.ServerName,
			&e.Source.SteamID, &e.Source.PersonaName, &e.Source.AvatarFull, &e.Source.Avatar,
			&e.Target.SteamID, &e.Target.PersonaName, &e.Target.AvatarFull, &e.Target.Avatar); err2 != nil {
			return nil, err2
		}
		events = append(events, e)
	}
	return events, nil
}

// TODO dont treat all origin positions as invalid
func (db *pgStore) BatchInsertServerLogs(ctx context.Context, logs []model.ServerEvent) error {
	const (
		stmtName = "insert-log"
		query    = `
		INSERT INTO server_log (
		    server_id, event_type, source_id, target_id, created_on, weapon, damage, 
		    item, extra, player_class, attacker_position, victim_position, assister_position
		) VALUES (
		    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 
		    CASE WHEN $11 != 0 AND $12 != 0 AND $13 != 0 THEN
		    	ST_SetSRID(ST_MakePoint($11, $12, $13), 4326)
		    END,
		    CASE WHEN $14 != 0 AND $15 != 0 AND $16 != 0 THEN
		    	ST_SetSRID(ST_MakePoint($14, $15, $16), 4326)
			END,
		    CASE WHEN $17 != 0 AND $18 != 0 AND $19 != 0 THEN
		          ST_SetSRID(ST_MakePoint($17, $18, $19), 4326)
			END)`
	)
	tx, err := db.c.Begin(ctx)
	if err != nil {
		return errors.Wrapf(err, "Failed to prepare logWriter query: %v", err)
	}
	_, errP := tx.Prepare(ctx, stmtName, query)
	if errP != nil {
		return errors.Wrapf(errP, "Failed to prepare logWriter query: %v", errP)
	}
	lCtx, cancel := context.WithTimeout(ctx, config.DB.LogWriteFreq/2)
	defer cancel()

	var re error
	for _, lg := range logs {
		if lg.Server == nil || lg.Server.ServerID <= 0 {
			continue
		}
		source := steamid.SID64(0)
		target := steamid.SID64(0)
		if lg.Source != nil && lg.Source.SteamID.Valid() {
			source = lg.Source.SteamID
		}
		if lg.Target != nil && lg.Target.SteamID.Valid() {
			target = lg.Target.SteamID
		}

		if _, re = tx.Exec(lCtx, stmtName, lg.Server.ServerID, lg.EventType,
			source.Int64(), target.Int64(), lg.CreatedOn, lg.Weapon, lg.Damage,
			lg.Item, lg.Extra, lg.PlayerClass,
			lg.AttackerPOS.Y, lg.AttackerPOS.X, lg.AttackerPOS.Z,
			lg.VictimPOS.Y, lg.VictimPOS.X, lg.VictimPOS.Z,
			lg.AssisterPOS.Y, lg.AssisterPOS.X, lg.AssisterPOS.Z); re != nil {
			re = errors.Wrapf(re, "Failed to write log entries")
			break
		}
	}
	if re != nil {
		if errR := tx.Rollback(lCtx); errR != nil {
			return errors.Wrapf(errR, "BatchInsertServerLogs rollback failed")
		}
		return re
	}
	if errC := tx.Commit(lCtx); errC != nil {
		log.Errorf("Failed to commit log entries: %v", errC)
	}
	return nil
}
