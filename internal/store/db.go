package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// ErrNoResult is returned on successful queries which return no rows
	ErrNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row value is attempted to be inserted
	ErrDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	//go:embed migrations
	migrations embed.FS
)

type tableName string

const (
	tableBan tableName = "ban"
	//tableBanAppeal    tableName = "ban_appeal"
	tableBanNet       tableName = "ban_net"
	tableFilteredWord tableName = "filtered_word"
	tableNetLocation  tableName = "net_location"
	tableNetProxy     tableName = "net_proxy"
	tableNetASN       tableName = "net_asn"
	//tablePerson       tableName = "person"
	tablePersonIP tableName = "person_ip"
	//tablePersonNames  tableName = "person_names"
	tableServer tableName = "server"
	//tableServerLog tableName = "server_log"
)

// QueryFilter provides a structure for common query parameters
type QueryFilter struct {
	Offset   uint64 `json:"offset" uri:"offset" binding:"gte=0"`
	Limit    uint64 `json:"limit" uri:"limit" binding:"gte=0,lte=1000"`
	SortDesc bool   `json:"desc" uri:"desc"`
	Query    string `json:"query" uri:"query"`
	OrderBy  string `json:"order_by" uri:"order_by"`
}

func (qf *QueryFilter) orderString() string {
	dir := "DESC"
	if !qf.SortDesc {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", qf.OrderBy, dir)
}

func NewQueryFilter(query string) *QueryFilter {
	return &QueryFilter{
		Limit:    1000,
		Offset:   0,
		SortDesc: true,
		OrderBy:  "created_on",
		Query:    query,
	}
}

// New sets up underlying required services.
func New(dsn string) (Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}
	ndb := PGStore{}
	if config.DB.AutoMigrate {
		if errM := ndb.Migrate(MigrateUp); errM != nil {
			if errM.Error() == "no change" {
				log.Infof("Migration at latest version")
			} else {
				log.Fatalf("Could not migrate schema: %v", errM)
			}
		} else {
			log.Infof("Migration completed successfully")
		}
	}
	if config.DB.LogQueries {
		lgr := log.New()
		lvl, err2 := log.ParseLevel(config.Log.Level)
		if err2 != nil {
			log.Fatalf("Invalid log level: %s (%v)", config.Log.Level, err2)
		}
		lgr.SetLevel(lvl)
		lgr.SetFormatter(&log.TextFormatter{
			ForceColors:   config.Log.ForceColours,
			DisableColors: config.Log.DisableColours,
			FullTimestamp: config.Log.FullTimestamp,
		})
		lgr.SetReportCaller(config.Log.ReportCaller)
		cfg.ConnConfig.Logger = logrusadapter.NewLogger(lgr)
	}
	dbConn, err3 := pgxpool.ConnectConfig(context.Background(), cfg)
	if err3 != nil {
		log.Fatalf("Failed to connect to database: %v", err3)
	}
	return &PGStore{
		c:             dbConn,
		cacheServerMu: &sync.RWMutex{},
		cacheServer:   map[int64]*model.Server{},
	}, nil
}

type PGStore struct {
	c             *pgxpool.Pool
	cacheServerMu *sync.RWMutex
	cacheServer   map[int64]*model.Server
}

func (db *PGStore) Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

var columnsServer = []string{"server_id", "short_name", "token", "address", "port", "rcon", "password",
	"token_created_on", "created_on", "updated_on", "reserved_slots"}

func (db *PGStore) GetServer(ctx context.Context, serverID int64, s *model.Server) error {
	var found bool
	db.cacheServerMu.RLock()
	s, found = db.cacheServer[serverID]
	db.cacheServerMu.RUnlock()
	if found {
		return nil
	}
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
			&s.ReservedSlots); err != nil {
		return dbErr(err)
	}

	db.cacheServerMu.Lock()
	db.cacheServer[serverID] = s
	db.cacheServerMu.Unlock()

	return nil
}

func (db *PGStore) GetServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	q, _, e := sb.Select(columnsServer...).
		From(string(tableServer)).
		ToSql()
	if e != nil {
		return nil, e
	}
	rows, err := db.c.Query(ctx, q)
	if err != nil {
		return []model.Server{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.Server
		if err2 := rows.Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port, &s.RCON,
			&s.Password, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err2 != nil {
			return nil, err2
		}
		servers = append(servers, s)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return servers, nil
}

func (db *PGStore) GetServerByName(ctx context.Context, serverName string, s *model.Server) error {
	db.cacheServerMu.RLock()
	for _, srv := range db.cacheServer {
		if srv.ServerName == serverName {
			db.cacheServerMu.RUnlock()
			return nil
		}
	}
	db.cacheServerMu.RUnlock()
	q, a, e := sb.Select("server_id", "short_name", "token", "address", "port", "rcon",
		"token_created_on", "created_on", "updated_on", "reserved_slots", "password").
		From(string(tableServer)).
		Where(sq.Eq{"short_name": serverName}).
		ToSql()
	if e != nil {
		return e
	}
	if err := db.c.QueryRow(ctx, q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn,
			&s.ReservedSlots, &s.Password); err != nil {
		return err
	}
	db.cacheServerMu.Lock()
	db.cacheServer[s.ServerID] = s
	db.cacheServerMu.Unlock()
	return nil
}

// SaveServer updates or creates the server data in the database
func (db *PGStore) SaveServer(ctx context.Context, server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return db.updateServer(ctx, server)
	}
	server.CreatedOn = config.Now()
	return db.insertServer(ctx, server)
}

func (db *PGStore) insertServer(ctx context.Context, s *model.Server) error {
	q, a, e := sb.Insert(string(tableServer)).
		Columns("short_name", "token", "address", "port",
			"rcon", "token_created_on", "created_on", "updated_on", "password", "reserved_slots").
		Values(s.ServerName, s.Token, s.Address, s.Port, s.RCON, s.TokenCreatedOn,
			s.CreatedOn, s.UpdatedOn, s.Password, s.ReservedSlots).
		Suffix("RETURNING server_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&s.ServerID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) updateServer(ctx context.Context, s *model.Server) error {
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
		Where(sq.Eq{"server_id": s.ServerID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update s")
	}
	db.cacheServerMu.Lock()
	delete(db.cacheServer, s.ServerID)
	db.cacheServerMu.Unlock()
	return nil
}

func (db *PGStore) DropServer(ctx context.Context, serverID int64) error {
	q, a, e := sb.Delete(string(tableServer)).Where(sq.Eq{"server_id": serverID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return err
	}
	db.cacheServerMu.Lock()
	delete(db.cacheServer, serverID)
	db.cacheServerMu.Unlock()
	return nil
}

func (db *PGStore) DropBan(ctx context.Context, ban *model.Ban) error {
	q, a, e := sb.Delete(string(tableBan)).Where(sq.Eq{"ban_id": ban.BanID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) getBanByColumn(ctx context.Context, column string, identifier interface{}, full bool, b *model.BannedPerson) error {
	q, a, e := sb.Select(
		"b.ban_id", "b.steam_id", "b.author_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.ban_source", "b.valid_until", "b.created_on", "b.updated_on",
		"p.steam_id as sid2", "p.created_on as created_on2", "p.updated_on as updated_on2", "p.communityvisibilitystate",
		"p.profilestate",
		"p.personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull", "p.avatarhash",
		"p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode", "p.loccityid",
		"p.permission_level", "p.discord_id", "p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban",
		"p.days_since_last_ban").
		From(fmt.Sprintf("%s b", tableBan)).
		LeftJoin("person p ON b.steam_id = p.steam_id").
		GroupBy("b.ban_id, p.steam_id").
		Where(sq.And{sq.Eq{fmt.Sprintf("b.%s", column): identifier}, sq.Gt{"b.valid_until": config.Now()}}).
		OrderBy("b.created_on DESC").
		Limit(1).
		ToSql()
	if e != nil {
		return e
	}
	if err := db.c.QueryRow(ctx, q, a...).
		Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName,
			&b.Person.ProfileURL, &b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull,
			&b.Person.AvatarHash, &b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID, &b.Person.PermissionLevel, &b.Person.DiscordID, &b.Person.CommunityBanned,
			&b.Person.VACBans, &b.Person.GameBans, &b.Person.EconomyBan, &b.Person.DaysSinceLastBan); err != nil {
		return dbErr(err)
	}
	if full {
		h, err := db.GetChatHistory(ctx, b.Person.SteamID)
		if err == nil {
			b.HistoryChat = h
		}
		b.HistoryConnections = []string{}
		ips, _ := db.GetIPHistory(ctx, b.Person.SteamID)
		b.HistoryIP = ips
		b.HistoryPersonaName = []string{}
	}
	return nil
}

func (db *PGStore) GetBanBySteamID(ctx context.Context, steamID steamid.SID64, full bool, p *model.BannedPerson) error {
	return db.getBanByColumn(ctx, "steam_id", steamID, full, p)
}

func (db *PGStore) GetBanByBanID(ctx context.Context, banID uint64, full bool, p *model.BannedPerson) error {
	return db.getBanByColumn(ctx, "ban_id", banID, full, p)
}

func (db *PGStore) GetChatHistory(ctx context.Context, sid64 steamid.SID64) ([]logparse.SayEvt, error) {
	const q = `
		SELECT l.source_id, coalesce(p.personaname, ''), l.extra
		FROM server_log l
		LEFT JOIN person p on l.source_id = p.steam_id
		WHERE source_id = $1
		  AND (event_type = 10 OR event_type = 11) 
		ORDER BY l.created_on DESC`
	rows, err := db.c.Query(ctx, q, sid64.String())
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	var hist []logparse.SayEvt
	for rows.Next() {
		var h logparse.SayEvt
		if err2 := rows.Scan(&h.SourcePlayer.SID, &h.SourcePlayer.Name, &h.Msg); err2 != nil {
			return nil, err2
		}
		hist = append(hist, h)
	}
	return hist, nil
}

func (db *PGStore) FindLogEvents(ctx context.Context, opts model.LogQueryOpts) ([]model.ServerEvent, error) {
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

func (db *PGStore) AddPersonIP(ctx context.Context, p *model.Person, ip string) error {
	q, a, e := sb.Insert(string(tablePersonIP)).
		Columns("steam_id", "ip_addr", "created_on").
		Values(p.SteamID, ip, config.Now()).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.c.Exec(ctx, q, a...)
	return dbErr(err)
}

func (db *PGStore) GetIPHistory(ctx context.Context, sid64 steamid.SID64) ([]model.PersonIPRecord, error) {
	q, a, e := sb.Select("ip_addr", "created_on").
		From(string(tablePersonIP)).
		Where(sq.Eq{"steam_id": sid64}).
		OrderBy("created_on DESC").
		ToSql()
	if e != nil {
		return nil, e
	}
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	var records []model.PersonIPRecord
	for rows.Next() {
		var r model.PersonIPRecord
		if err2 := rows.Scan(&r.IP, &r.CreatedOn); err2 != nil {
			return nil, dbErr(err)
		}
		records = append(records, r)
	}
	return records, nil
}

func (db *PGStore) GetAppeal(ctx context.Context, banID uint64, ap *model.Appeal) error {
	q, a, e := sb.Select("appeal_id", "ban_id", "appeal_text", "appeal_state",
		"email", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.Eq{"ban_id": banID}).
		ToSql()
	if e != nil {
		return e
	}

	if err := db.c.QueryRow(ctx, q, a...).
		Scan(&ap.AppealID, &ap.BanID, &ap.AppealText, &ap.AppealState, &ap.Email, &ap.CreatedOn,
			&ap.UpdatedOn); err != nil {
		return err
	}
	return nil
}

func (db *PGStore) updateAppeal(ctx context.Context, appeal *model.Appeal) error {
	q, a, e := sb.Update("ban_appeal").
		Set("appeal_text", appeal.AppealText).
		Set("appeal_state", appeal.AppealState).
		Set("email", appeal.Email).
		Set("updated_on", appeal.UpdatedOn).
		Where(sq.Eq{"appeal_id": appeal.AppealID}).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.c.Exec(ctx, q, a...)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) insertAppeal(ctx context.Context, ap *model.Appeal) error {
	q, a, e := sb.Insert("ban_appeal").
		Columns("ban_id", "appeal_text", "appeal_state", "email", "created_on", "updated_on").
		Values(ap.BanID, ap.AppealText, ap.AppealState, ap.Email, ap.CreatedOn, ap.UpdatedOn).
		Suffix("RETURNING appeal_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&ap.AppealID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) SaveAppeal(ctx context.Context, appeal *model.Appeal) error {
	appeal.UpdatedOn = config.Now()
	if appeal.AppealID > 0 {
		return db.updateAppeal(ctx, appeal)
	}
	appeal.CreatedOn = config.Now()
	return db.insertAppeal(ctx, appeal)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically
func (db *PGStore) SaveBan(ctx context.Context, ban *model.Ban) error {
	// Ensure the foreign keys are satisfied
	var p model.Person
	err := db.GetOrCreatePersonBySteamID(ctx, ban.SteamID, &p)
	if err != nil {
		return errors.Wrapf(err, "Failed to get person for ban")
	}
	var a model.Person
	err2 := db.GetOrCreatePersonBySteamID(ctx, ban.AuthorID, &a)
	if err2 != nil {
		return errors.Wrapf(err, "Failed to get author for ban")
	}
	ban.UpdatedOn = config.Now()
	if ban.BanID > 0 {
		return db.updateBan(ctx, ban)
	}
	ban.CreatedOn = config.Now()
	var existing model.BannedPerson
	e := db.GetBanBySteamID(ctx, ban.SteamID, false, &existing)
	if e != nil && !errors.Is(e, ErrNoResult) {
		return errors.Wrapf(err, "Failed to check existing ban state")
	}
	if ban.BanType <= existing.Ban.BanType {
		return ErrDuplicate
	}
	return db.insertBan(ctx, ban)
}

func (db *PGStore) insertBan(ctx context.Context, ban *model.Ban) error {
	q, a, e := sb.Insert("ban").
		Columns("steam_id", "author_id", "ban_type", "reason", "reason_text",
			"note", "valid_until", "created_on", "updated_on", "ban_source").
		Values(ban.SteamID, ban.AuthorID, ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Source).
		Suffix("RETURNING ban_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&ban.BanID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) updateBan(ctx context.Context, ban *model.Ban) error {
	q, a, e := sb.Update("ban").
		Set("author_id", ban.AuthorID).
		Set("ban_type", ban.BanType).
		Set("reason", ban.Reason).
		Set("reason_text", ban.ReasonText).
		Set("note", ban.Note).
		Set("valid_until", ban.ValidUntil).
		Set("updated_on", ban.UpdatedOn).
		Set("ban_source", ban.Source).
		Where(sq.Eq{"ban_id": ban.BanID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	q, a, e := sb.Delete("person").Where(sq.Eq{"steam_id": steamID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

// SavePerson will insert or update the person record
func (db *PGStore) SavePerson(ctx context.Context, person *model.Person) error {
	person.UpdatedOn = config.Now()
	if !person.IsNew {
		return db.updatePerson(ctx, person)
	}
	person.CreatedOn = person.UpdatedOn
	return db.insertPerson(ctx, person)
}

func (db *PGStore) updatePerson(ctx context.Context, p *model.Person) error {
	p.UpdatedOn = config.Now()
	q, a, e := sb.Update("person").
		Set("updated_on", p.UpdatedOn).
		Set("communityvisibilitystate", p.CommunityVisibilityState).
		Set("profilestate", p.ProfileState).
		Set("personaname", p.PersonaName).
		Set("profileurl", p.ProfileURL).
		Set("avatar", p.Avatar).
		Set("avatarmedium", p.AvatarMedium).
		Set("avatarfull", p.PlayerSummary.AvatarFull).
		Set("avatarhash", p.PlayerSummary.AvatarHash).
		Set("personastate", p.PlayerSummary.PersonaState).
		Set("realname", p.PlayerSummary.RealName).
		Set("timecreated", p.PlayerSummary.TimeCreated).
		Set("loccountrycode", p.PlayerSummary.LocCountryCode).
		Set("locstatecode", p.PlayerSummary.LocStateCode).
		Set("loccityid", p.PlayerSummary.LocCityID).
		Set("permission_level", p.PermissionLevel).
		Set("discord_id", p.DiscordID).
		Set("community_banned", p.CommunityBanned).
		Set("vac_bans", p.VACBans).
		Set("game_bans", p.GameBans).
		Set("economy_ban", p.EconomyBan).
		Set("days_since_last_ban", p.DaysSinceLastBan).
		Where(sq.Eq{"steam_id": p.SteamID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) insertPerson(ctx context.Context, p *model.Person) error {
	q, a, e := sb.
		Insert("person").
		Columns(
			"created_on", "updated_on", "steam_id", "communityvisibilitystate",
			"profilestate", "personaname", "profileurl", "avatar", "avatarmedium", "avatarfull",
			"avatarhash", "personastate", "realname", "timecreated", "loccountrycode", "locstatecode",
			"loccityid", "permission_level", "discord_id", "community_banned", "vac_bans", "game_bans",
			"economy_ban", "days_since_last_ban").
		Values(p.CreatedOn, p.UpdatedOn, p.SteamID,
			p.CommunityVisibilityState, p.ProfileState, p.PersonaName, p.ProfileURL,
			p.Avatar, p.AvatarMedium, p.AvatarFull, p.AvatarHash, p.PersonaState, p.RealName, p.TimeCreated,
			p.LocCountryCode, p.LocStateCode, p.LocCityID, p.PermissionLevel, p.DiscordID, p.CommunityBanned, p.VACBans,
			p.GameBans, p.EconomyBan, p.DaysSinceLastBan).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.c.Exec(ctx, q, a...)
	if err != nil {
		return dbErr(err)
	}
	p.IsNew = false
	return nil
}

//"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban"
var profileColumns = []string{"steam_id", "created_on", "updated_on",
	"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
	"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
	"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban"}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func (db *PGStore) GetPersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error {
	const q = `
    WITH addresses as (
		SELECT steam_id, ip_addr FROM person_ip
		WHERE steam_id = $1
		ORDER BY created_on DESC limit 1
	)
	SELECT 
	    p.steam_id, created_on, updated_on, communityvisibilitystate, profilestate, personaname, profileurl, avatar,
		avatarmedium, avatarfull, avatarhash, personastate, realname, timecreated, loccountrycode, locstatecode, loccityid,
		permission_level, discord_id, a.ip_addr, community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban 
	FROM person p
	left join addresses a on p.steam_id = a.steam_id
	WHERE p.steam_id = $1;`

	p.IsNew = false
	p.PlayerSummary = &steamweb.PlayerSummary{}
	err := db.c.QueryRow(ctx, q, sid.Int64()).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.IPAddr, &p.CommunityBanned,
		&p.VACBans, &p.GameBans, &p.EconomyBan, &p.DaysSinceLastBan)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) GetPeople(ctx context.Context, qf *QueryFilter) ([]*model.Person, error) {
	qb := sb.Select(profileColumns...).From("person")
	if qf.Query != "" {
		// TODO add lower-cased functional index to avoid tableName scan
		qb = qb.Where(sq.ILike{"personaname": strings.ToLower(qf.Query)})
	}
	if qf.Offset > 0 {
		qb = qb.Offset(qf.Offset)
	}
	if qf.OrderBy != "" {
		qb = qb.OrderBy(qf.orderString())
	}
	if qf.Limit == 0 {
		qb = qb.Limit(100)
	} else {
		qb = qb.Limit(qf.Limit)
	}
	q, a, e := qb.ToSql()
	if e != nil {
		return nil, e
	}
	var people []*model.Person
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	for rows.Next() {
		p := model.NewPerson(0)
		if err2 := rows.Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.IPAddr, &p.CommunityVisibilityState,
			&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
			&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID,
			&p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans, &p.EconomyBan,
			&p.DaysSinceLastBan); err2 != nil {
			return nil, err2
		}
		people = append(people, &p)
	}
	return people, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func (db *PGStore) GetOrCreatePersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error {
	err := db.GetPersonBySteamID(ctx, sid, p)
	if err != nil && dbErr(err) == ErrNoResult {
		// FIXME
		//p = model.NewPerson(sid)
		p.SteamID = sid
		return db.SavePerson(ctx, p)
	} else if err != nil {
		return err
	}
	return nil
}

// GetPersonByDiscordID returns a person by their discord_id
func (db *PGStore) GetPersonByDiscordID(ctx context.Context, did string, p *model.Person) error {
	q, a, e := sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": did}).
		ToSql()
	if e != nil {
		return e
	}
	p.IsNew = false
	p.PlayerSummary = &steamweb.PlayerSummary{}
	err := db.c.QueryRow(ctx, q, a...).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans,
		&p.EconomyBan, &p.DaysSinceLastBan)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

// GetBanNet returns the BanNet matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (db *PGStore) GetBanNet(ctx context.Context, ip net.IP) ([]model.BanNet, error) {
	q, _, e := sb.Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From("ban_net").
		Suffix("WHERE $1 <<= cidr").
		ToSql()
	if e != nil {
		return nil, e
	}
	var nets []model.BanNet
	rows, err := db.c.Query(ctx, q, ip.String())
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n model.BanNet
		if err2 := rows.Scan(&n.NetID, &n.CIDR, &n.Source, &n.CreatedOn, &n.UpdatedOn, &n.Reason, &n.ValidUntil); err2 != nil {
			return nil, err2
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func (db *PGStore) updateBanNet(ctx context.Context, banNet *model.BanNet) error {
	q, a, e := sb.Update("ban_net").
		Set("cidr", banNet.CIDR).
		Set("source", banNet.Source).
		Set("created_on", banNet.CreatedOn).
		Set("updated_on", banNet.UpdatedOn).
		Set("reason", banNet.Reason).
		Set("valid_until_id", banNet.ValidUntil).
		Where(sq.Eq{"net_id": banNet.NetID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return err
	}
	return nil
}

func (db *PGStore) insertBanNet(ctx context.Context, banNet *model.BanNet) error {
	q, a, e := sb.Insert("ban_net").
		Columns("cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		Values(banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ValidUntil).
		Suffix("RETURNING net_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&banNet.NetID)
	if err != nil {
		return err
	}
	return nil
}

func (db *PGStore) SaveBanNet(ctx context.Context, banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return db.updateBanNet(ctx, banNet)
	}
	return db.insertBanNet(ctx, banNet)
}

func (db *PGStore) DropNetBan(ctx context.Context, ban model.BanNet) error {
	q, a, e := sb.Delete("ban_net").Where(sq.Eq{"net_id": ban.NetID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) GetExpiredBans(ctx context.Context) ([]*model.Ban, error) {
	q, a, e := sb.Select(
		"ban_id", "steam_id", "author_id", "ban_type", "reason", "reason_text", "note",
		"valid_until", "ban_source", "created_on", "updated_on").
		From(string(tableBan)).
		Where(sq.Lt{"valid_until": config.Now()}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []*model.Ban
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err2 := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.ValidUntil, &b.Source, &b.CreatedOn, &b.UpdatedOn); err2 != nil {
			return nil, err2
		}
		bans = append(bans, &b)
	}
	return bans, nil
}

//func GetBansTotal(o *QueryFilter) (int, error) {
//	q, _, e := sb.Select("count(*) as total_rows").From(string(tableBan)).ToSql()
//	if e != nil {
//		return 0, e
//	}
//	var total int
//	if err := db.QueryRow(context.Background(), q).Scan(&total); err != nil {
//		return 0, err
//	}
//	return total, nil
//}

// GetBans returns all bans that fit the filter criteria passed in
func (db *PGStore) GetBans(ctx context.Context, o *QueryFilter) ([]*model.BannedPerson, error) {
	q, a, e := sb.Select(
		"b.ban_id", "b.steam_id", "b.author_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.ban_source", "b.valid_until", "b.created_on", "b.updated_on",
		"p.steam_id", "p.created_on", "p.updated_on", "p.communityvisibilitystate", "p.profilestate",
		"p.personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull", "p.avatarhash",
		"p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode", "p.loccityid",
		"p.permission_level", "p.discord_id", "p.community_banned", "p.vac_bans", "p.game_bans",
		"p.economy_ban", "p.days_since_last_ban").
		From(fmt.Sprintf("%s b", string(tableBan))).
		LeftJoin("person p on p.steam_id = b.steam_id").
		OrderBy(fmt.Sprintf("b.%s", o.OrderBy)).
		Limit(o.Limit).
		Offset(o.Offset).
		ToSql()

	if e != nil {
		return nil, errors.Wrapf(e, "Failed to execute: %s", q)
	}
	var bans []*model.BannedPerson
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		b := model.NewBannedPerson()
		if err := rows.Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName, &b.Person.ProfileURL,
			&b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull, &b.Person.AvatarHash,
			&b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID, &b.Person.PermissionLevel,
			&b.Person.DiscordID, &b.Person.CommunityBanned, &b.Person.VACBans, &b.Person.GameBans, &b.Person.EconomyBan,
			&b.Person.DaysSinceLastBan); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (db *PGStore) GetBansOlderThan(ctx context.Context, o *QueryFilter, t time.Time) ([]model.Ban, error) {
	q, a, e := sb.
		Select("ban_id", "steam_id", "author_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "created_on", "updated_on", "ban_source").
		From(string(tableBan)).
		Where(sq.Lt{"updated_on": t}).
		Limit(o.Limit).Offset(o.Offset).ToSql()
	if e != nil {
		return nil, e
	}
	var bans []model.Ban
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err = rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.ValidUntil, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (db *PGStore) GetExpiredNetBans(ctx context.Context) ([]model.BanNet, error) {
	q, a, e := sb.
		Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From(string(tableBanNet)).
		Where(sq.Lt{"valid_until": config.Now()}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []model.BanNet
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BanNet
		if err2 := rows.Scan(&b.NetID, &b.CIDR, &b.Source, &b.CreatedOn, &b.UpdatedOn, &b.Reason, &b.ValidUntil); err2 != nil {
			return nil, err2
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (db *PGStore) InsertFilter(ctx context.Context, rx string) (*model.Filter, error) {
	r, e := regexp.Compile(rx)
	if e != nil {
		return nil, e
	}
	filter := &model.Filter{
		Word:      r,
		CreatedOn: config.Now(),
	}
	q, a, e := sb.Insert(string(tableFilteredWord)).
		Columns("word", "created_on").
		Values(rx, filter.CreatedOn).
		Suffix("RETURNING word_id").
		ToSql()
	if e != nil {
		return nil, e
	}
	if err := db.c.QueryRow(ctx, q, a...).Scan(&filter.WordID); err != nil {
		return nil, dbErr(err)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return filter, nil
}

func (db *PGStore) DropFilter(ctx context.Context, filter *model.Filter) error {
	q, a, e := sb.Delete(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": filter.WordID}).
		ToSql()
	if e != nil {
		return dbErr(e)
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	log.Debugf("Deleted filter: %d", filter.WordID)
	return nil
}

func (db *PGStore) GetFilterByID(ctx context.Context, wordId int, f *model.Filter) error {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": wordId}).
		ToSql()
	if e != nil {
		return dbErr(e)
	}
	var w string
	if err := db.c.QueryRow(ctx, q, a...).Scan(&f.WordID, &w, &f.CreatedOn); err != nil {
		return errors.Wrapf(err, "Failed to load filter")
	}
	rx, er := regexp.Compile(w)
	if er != nil {
		return er
	}
	f.Word = rx
	return nil
}

func (db *PGStore) GetFilters(ctx context.Context) ([]*model.Filter, error) {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).ToSql()
	if e != nil {
		return nil, dbErr(e)
	}
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, dbErr(err)
	}
	var filters []*model.Filter
	defer rows.Close()
	for rows.Next() {
		var f model.Filter
		var w string
		if err = rows.Scan(&f.WordID, &w, &f.CreatedOn); err != nil {
			return nil, errors.Wrapf(err, "Failed to load filter")
		}
		rx, er := regexp.Compile(w)
		if er != nil {
			return nil, er
		}
		f.Word = rx
		filters = append(filters, &f)
	}
	return filters, nil
}

// TODO dont treat all origin positions as invalid
func (db *PGStore) BatchInsertServerLogs(ctx context.Context, logs []model.ServerEvent) error {
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

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func (db *PGStore) InsertBlockListData(ctx context.Context, d *ip2location.BlockListData) error {
	if len(d.Proxies) > 0 {
		if err := db.loadProxies(ctx, d.Proxies, false); err != nil {
			return err
		}
	}
	if len(d.Locations4) > 0 {
		if err := db.loadLocation(ctx, d.Locations4, false); err != nil {
			return err
		}
	}
	if len(d.ASN4) > 0 {
		if err := db.loadASN(ctx, d.ASN4); err != nil {
			return err
		}
	}
	return nil
}

func (db *PGStore) GetASNRecord(ctx context.Context, ip net.IP, r *ip2location.ASNRecord) error {
	q, _, e := sb.Select("ip_from", "ip_to", "cidr", "as_num", "as_name").
		From("net_asn").
		Where("$1 << cidr").
		Limit(1).
		ToSql()
	if e != nil {
		return e
	}
	if err := db.c.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.CIDR, &r.ASNum, &r.ASName); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) GetLocationRecord(ctx context.Context, ip net.IP, r *ip2location.LocationRecord) error {
	const q = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE $1 <@ ip_range`
	if err := db.c.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.LatLong.Latitude, &r.LatLong.Longitude); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) GetProxyRecord(ctx context.Context, ip net.IP, r *ip2location.ProxyRecord) error {
	const q = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1 <@ ip_range`
	if err := db.c.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.ProxyType, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.ISP,
			&r.Domain, &r.UsageType, &r.ASN, &r.AS, &r.LastSeen, &r.Threat); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) GetPersonIPHistory(ctx context.Context, sid steamid.SID64) ([]model.PersonIPRecord, error) {
	const q = `
		SELECT
			   ip.ip_addr, ip.created_on,
			   l.city_name, l.country_name, l.country_code,
			   a.as_name, a.as_num,
			   coalesce(p.isp, ''), coalesce(p.usage_type, ''), 
		       coalesce(p.threat, ''), coalesce(p.domain_used, '')
		FROM person_ip ip
		LEFT JOIN net_location l ON ip.ip_addr <@ l.ip_range
		LEFT JOIN net_asn a ON ip.ip_addr <@ a.ip_range
		LEFT OUTER JOIN net_proxy p ON ip.ip_addr <@ p.ip_range
		WHERE ip.steam_id = $1`
	rows, err := db.c.Query(ctx, q, sid.Int64())
	if err != nil {
		return nil, err
	}
	var records []model.PersonIPRecord
	defer rows.Close()
	for rows.Next() {
		var r model.PersonIPRecord
		if errR := rows.Scan(&r.IP, &r.CreatedOn, &r.CityName, &r.CountryName, &r.CountryCode, &r.ASName,
			&r.ASNum, &r.ISP, &r.UsageType, &r.Threat, &r.DomainUsed); errR != nil {
			return nil, errR
		}
		records = append(records, r)
	}
	return records, nil
}

func (db *PGStore) truncateTable(ctx context.Context, table tableName) error {
	if _, err := db.c.Exec(ctx, fmt.Sprintf("TRUNCATE %s;", table)); err != nil {
		return dbErr(err)
	}
	return nil
}

func (db *PGStore) loadASN(ctx context.Context, records []ip2location.ASNRecord) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetASN); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_asn (ip_from, ip_to, cidr, as_num, as_name, ip_range) 
		VALUES($1, $2, $3, $4, $5, iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.CIDR, a.ASNum, a.ASName)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("ASN Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d ASN4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (db *PGStore) loadLocation(ctx context.Context, records []ip2location.LocationRecord, _ bool) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetLocation); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_location (ip_from, ip_to, country_code, country_name, region_name, city_name, location, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($8, $7), 4326), iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.CountryCode, a.CountryName, a.RegionName, a.CityName, a.LatLong.Latitude, a.LatLong.Longitude)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("Location4 Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Location4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (db *PGStore) loadProxies(ctx context.Context, records []ip2location.ProxyRecord, _ bool) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetProxy); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_proxy (ip_from, ip_to, proxy_type, country_code, country_name, region_name, city_name, isp,
		                       domain_used, usage_type, as_num, as_name, last_seen, threat, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.ProxyType, a.CountryCode, a.CountryName, a.RegionName, a.CityName,
			a.ISP, a.Domain, a.UsageType, a.ASN, a.AS, a.LastSeen, a.Threat)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("Proxy Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Proxy records in %s", len(records), time.Since(t0).String())
	return nil
}

func (db *PGStore) GetStats(ctx context.Context, stats *model.Stats) error {
	const q = `
	SELECT 
		(SELECT COUNT(ban_id) FROM ban) as bans_total,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_day,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_week,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 MONTH')) as bans_month, 
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '3 MONTH')) as bans_3month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '6 MONTH')) as bans_6month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 YEAR')) as bans_year,
		(SELECT COUNT(net_id) FROM ban_net) as bans_cidr, 
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 0) as appeals_open,
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 1 OR appeal_state = 2) as appeals_closed,
		(SELECT COUNT(word_id) FROM filtered_word) as filtered_words,
		(SELECT COUNT(server_id) FROM server) as servers_total`
	if err := db.c.QueryRow(ctx, q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); err != nil {
		log.Errorf("Failed to fetch stats: %v", err)
		return dbErr(err)
	}
	return nil

}

// dbErr is used to wrap common database errors in own own error types
func dbErr(err error) error {
	if err == nil {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			log.Errorf("Unhandled store error: (%s) %s", pgErr.Code, pgErr.Message)
			return err
		}
	}
	if err.Error() == "no rows in result set" {
		return ErrNoResult
	}
	return err
}

// MigrationAction is the type of migration to perform
type MigrationAction int

const (
	// MigrateUp Fully upgrades the schema
	MigrateUp = iota
	// MigrateDn Fully downgrades the schema
	MigrateDn
	// MigrateUpOne Upgrade the schema by one revision
	// MigrateUpOne
	// MigrateDownOne Downgrade the schema by one revision
	// MigrateDownOne
)

// Migrate e
func (db *PGStore) Migrate(action MigrationAction) error {
	instance, err := sql.Open("pgx", config.DB.DSN)
	if err != nil {
		return errors.Wrapf(err, "Failed to open database for migration")
	}
	if errPing := instance.Ping(); errPing != nil {
		return errors.Wrapf(errPing, "Cannot migrate, failed to connect to target server")
	}
	driver, err2 := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      60 * time.Second,
		MultiStatementEnabled: true,
	})
	if err2 != nil {
		return errors.Wrapf(err2, "failed to create migration driver")
	}
	defer func() {
		if e := driver.Close(); e != nil {
			log.Errorf("Failed to close migrate driver: %v", e)
		}
	}()
	source, err3 := httpfs.New(http.FS(migrations), "migrations")
	if err3 != nil {
		return err3
	}
	m, err4 := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if err4 != nil {
		return errors.Wrapf(err4, "Failed to migrate up")
	}
	switch action {
	//case MigrateUpOne:
	//	return m.Steps(1)
	case MigrateDn:
		return m.Down()
	//case MigrateDownOne:
	//	return m.Steps(-1)
	case MigrateUp:
		fallthrough
	default:
		return m.Up()
	}
}

// Import will import bans from a root folder.
// The formatting is JSON with the importedBan schema defined inline
// Valid filenames are: main_ban.json
func (db *PGStore) Import(ctx context.Context, root string) error {
	type importedBan struct {
		BanID      int    `json:"ban_id"`
		SteamID    uint64 `json:"steam_id"`
		AuthorID   uint64 `json:"author_id"`
		BanType    int    `json:"ban_type"`
		Reason     int    `json:"reason"`
		ReasonText string `json:"reason_text"`
		Note       string `json:"note"`
		Until      int    `json:"until"`
		CreatedOn  int    `json:"created_on"`
		UpdatedOn  int    `json:"updated_on"`
		BanSource  int    `json:"ban_source"`
	}

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		switch d.Name() {
		case "main_ban.json":
			b, err := ioutil.ReadFile(path.Join(root, d.Name()))
			if err != nil {
				return err
			}
			var imported []importedBan
			if err2 := json.Unmarshal(b, &imported); err2 != nil {
				return err2
			}
			for _, im := range imported {
				var (
					b1 model.Person
					b2 model.Person
				)
				if e1 := db.GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.SteamID), &b1); e1 != nil {
					return e1
				}
				if e2 := db.GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.AuthorID), &b2); e2 != nil {
					return e2
				}
				sum, err3 := steamweb.PlayerSummaries(steamid.Collection{b1.SteamID, b2.SteamID})
				if err3 != nil {
					log.Errorf("Failed to get player summary: %v", err3)
					return err3
				}
				if len(sum) > 0 {
					b1.PlayerSummary = &sum[0]
					if err4 := db.SavePerson(ctx, &b1); err4 != nil {
						return err4
					}
					if b2.SteamID.Valid() && len(sum) > 1 {
						b2.PlayerSummary = &sum[1]
						if err5 := db.SavePerson(ctx, &b2); err5 != nil {
							return err5
						}
					}
				}
				bn := model.NewBan(b1.SteamID, b2.SteamID, 0)
				bn.ValidUntil = time.Unix(int64(im.Until), 0)
				bn.ReasonText = im.ReasonText
				bn.CreatedOn = time.Unix(int64(im.CreatedOn), 0)
				bn.UpdatedOn = time.Unix(int64(im.UpdatedOn), 0)
				bn.Source = model.System
				if err4 := db.SaveBan(ctx, &bn); err4 != nil {
					return err4
				}
			}
		}
		return nil
	})
}
