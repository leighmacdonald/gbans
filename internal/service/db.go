package service

import (
	"context"
	"database/sql"
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
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
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
	db           *pgxpool.Pool
	errNoResult  = errors.New("No results found")
	errDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries
	sb            = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	cacheServerMu *sync.RWMutex
	cacheServer   map[int64]model.Server
)

func init() {
	cacheServerMu = &sync.RWMutex{}
	cacheServer = map[int64]model.Server{}
}

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
	tableServer    tableName = "server"
	tableServerLog tableName = "server_log"
)

// queryFilter provides a structure for common query parameters
type queryFilter struct {
	Offset   uint64 `json:"offset" uri:"offset" binding:"gte=0"`
	Limit    uint64 `json:"limit" uri:"limit" binding:"gte=0,lte=1000"`
	SortDesc bool   `json:"desc" uri:"desc"`
	Query    string `json:"query" uri:"query"`
	OrderBy  string `json:"order_by" uri:"order_by"`
}

func (qf *queryFilter) orderString() string {
	dir := "DESC"
	if !qf.SortDesc {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", qf.OrderBy, dir)
}

func newQueryFilter(query string) *queryFilter {
	return &queryFilter{
		Limit:    1000,
		Offset:   0,
		SortDesc: true,
		OrderBy:  "created_on",
		Query:    query,
	}
}

// Init sets up underlying required services.
func Init(dsn string) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}
	if config.DB.AutoMigrate {
		if errM := Migrate(MigrateUp); errM != nil {
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

	db = dbConn
}

var columnsServer = []string{"server_id", "short_name", "token", "address", "port", "rcon", "password",
	"token_created_on", "created_on", "updated_on", "reserved_slots"}

func getServer(ctx context.Context, serverID int64) (model.Server, error) {
	var s model.Server
	var found bool
	cacheServerMu.RLock()
	s, found = cacheServer[serverID]
	cacheServerMu.RUnlock()
	if found {
		return s, nil
	}
	q, a, e := sb.Select(columnsServer...).
		From(string(tableServer)).
		Where(sq.Eq{"server_id": serverID}).
		ToSql()
	if e != nil {
		return model.Server{}, e
	}
	if err := db.QueryRow(ctx, q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port, &s.RCON,
			&s.Password, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn,
			&s.ReservedSlots); err != nil {
		return model.Server{}, dbErr(err)
	}
	cacheServerMu.Lock()
	cacheServer[serverID] = s
	cacheServerMu.Unlock()
	return s, nil
}

func getServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	q, _, e := sb.Select(columnsServer...).
		From(string(tableServer)).
		ToSql()
	if e != nil {
		return nil, e
	}
	rows, err := db.Query(ctx, q)
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

func getServerByName(ctx context.Context, serverName string) (model.Server, error) {
	serverStateMu.RLock()
	for _, srv := range cacheServer {
		if srv.ServerName == serverName {
			serverStateMu.RUnlock()
			return srv, nil
		}
	}
	serverStateMu.RUnlock()
	var s model.Server
	q, a, e := sb.Select("server_id", "short_name", "token", "address", "port", "rcon",
		"token_created_on", "created_on", "updated_on", "reserved_slots").
		From(string(tableServer)).
		Where(sq.Eq{"short_name": serverName}).
		ToSql()
	if e != nil {
		return model.Server{}, e
	}
	if err := db.QueryRow(ctx, q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
		return model.Server{}, err
	}
	serverStateMu.Lock()
	cacheServer[s.ServerID] = s
	serverStateMu.Unlock()
	return s, nil
}

// SaveServer updates or creates the server data in the database
func SaveServer(ctx context.Context, server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return updateServer(ctx, server)
	}
	server.CreatedOn = config.Now()
	return insertServer(ctx, server)
}

func insertServer(ctx context.Context, s *model.Server) error {
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
	err := db.QueryRow(ctx, q, a...).Scan(&s.ServerID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func updateServer(ctx context.Context, s *model.Server) error {
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
		Where(sq.Eq{"server_id": s.ServerID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update s")
	}
	serverStateMu.Lock()
	delete(cacheServer, s.ServerID)
	serverStateMu.Unlock()
	return nil
}

func dropServer(ctx context.Context, serverID int64) error {
	q, a, e := sb.Delete(string(tableServer)).Where(sq.Eq{"server_id": serverID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return err
	}
	cacheServerMu.Lock()
	delete(cacheServer, serverID)
	cacheServerMu.Unlock()
	return nil
}

func dropBan(ctx context.Context, ban *model.Ban) error {
	q, a, e := sb.Delete(string(tableBan)).Where(sq.Eq{"ban_id": ban.BanID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func getBanByColumn(ctx context.Context, column string, identifier interface{}, full bool) (*model.BannedPerson, error) {
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
	b := model.NewBannedPerson()
	if e != nil {
		return b, e
	}
	if err := db.QueryRow(ctx, q, a...).
		Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName,
			&b.Person.ProfileURL, &b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull,
			&b.Person.AvatarHash, &b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID, &b.Person.PermissionLevel, &b.Person.DiscordID, &b.Person.CommunityBanned,
			&b.Person.VACBans, &b.Person.GameBans, &b.Person.EconomyBan, &b.Person.DaysSinceLastBan); err != nil {
		return b, dbErr(err)
	}
	if full {
		h, err := getChatHistory(ctx, b.Person.SteamID)
		if err == nil {
			b.HistoryChat = h
		}
		b.HistoryConnections = []string{}
		b.HistoryIP = getIPHistory(ctx, b.Person.SteamID)
		b.HistoryPersonaName = []string{}
	}
	return b, nil
}

func getBanBySteamID(ctx context.Context, steamID steamid.SID64, full bool) (*model.BannedPerson, error) {
	return getBanByColumn(ctx, "steam_id", steamID, full)
}

func getBanByBanID(ctx context.Context, banID uint64, full bool) (*model.BannedPerson, error) {
	return getBanByColumn(ctx, "ban_id", banID, full)
}

func getChatHistory(ctx context.Context, sid64 steamid.SID64) ([]model.ChatLog, error) {
	q, a, e := sb.Select("payload -> message", "created_on").
		From(string(tableServerLog)).
		Where(sq.And{
			sq.Eq{"source_id": sid64},
			sq.Eq{"event_type": []logparse.MsgType{logparse.Say, logparse.SayTeam}},
		}).
		ToSql()
	if e != nil {
		return nil, e
	}
	rows, err := db.Query(ctx, q, a...)
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	var hist []model.ChatLog
	for rows.Next() {
		var h model.ChatLog
		if err2 := rows.Scan(&h.Message, h.CreatedOn); err2 != nil {
			return nil, err2
		}
		hist = append(hist, h)
	}
	return hist, nil
}

func addPersonIP(ctx context.Context, p *model.Person, ip string) error {
	q, a, e := sb.Insert(string(tablePersonIP)).
		Columns("steam_id", "ip_addr", "created_on").
		Values(p.SteamID, ip, config.Now()).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.Exec(ctx, q, a...)
	return dbErr(err)
}

func getIPHistory(ctx context.Context, sid64 steamid.SID64) []model.IPRecord {
	q, a, e := sb.Select("ip_addr", "created_on").
		From(string(tablePersonIP)).
		Where(sq.Eq{"steam_id": sid64}).
		OrderBy("created_on DESC").
		ToSql()
	if e != nil {
		return nil
	}
	rows, err := db.Query(ctx, q, a...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var records []model.IPRecord
	for rows.Next() {
		var r model.IPRecord
		if err2 := rows.Scan(&r.IPAddr, &r.CreatedOn); err2 != nil {
			return nil
		}
		records = append(records, r)
	}
	return records
}

func getAppeal(ctx context.Context, banID uint64) (model.Appeal, error) {
	q, a, e := sb.Select("appeal_id", "ban_id", "appeal_text", "appeal_state",
		"email", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.Eq{"ban_id": banID}).
		ToSql()
	if e != nil {
		return model.Appeal{}, e
	}
	var ap model.Appeal
	if err := db.QueryRow(ctx, q, a...).
		Scan(&ap.AppealID, &ap.BanID, &ap.AppealText, &ap.AppealState, &ap.Email, &ap.CreatedOn,
			&ap.UpdatedOn); err != nil {
		return model.Appeal{}, err
	}
	return ap, nil
}

func updateAppeal(ctx context.Context, appeal *model.Appeal) error {
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
	_, err := db.Exec(ctx, q, a...)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func insertAppeal(ctx context.Context, ap *model.Appeal) error {
	q, a, e := sb.Insert("ban_appeal").
		Columns("ban_id", "appeal_text", "appeal_state", "email", "created_on", "updated_on").
		Values(ap.BanID, ap.AppealText, ap.AppealState, ap.Email, ap.CreatedOn, ap.UpdatedOn).
		Suffix("RETURNING appeal_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.QueryRow(ctx, q, a...).Scan(&ap.AppealID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func saveAppeal(ctx context.Context, appeal *model.Appeal) error {
	appeal.UpdatedOn = config.Now()
	if appeal.AppealID > 0 {
		return updateAppeal(ctx, appeal)
	}
	appeal.CreatedOn = config.Now()
	return insertAppeal(ctx, appeal)
}

// saveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically
func saveBan(ctx context.Context, ban *model.Ban) error {
	// Ensure the foreign keys are satisfied
	_, err := GetOrCreatePersonBySteamID(ctx, ban.SteamID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get person for ban")
	}
	_, err2 := GetOrCreatePersonBySteamID(ctx, ban.AuthorID)
	if err2 != nil {
		return errors.Wrapf(err, "Failed to get author for ban")
	}
	ban.UpdatedOn = config.Now()
	if ban.BanID > 0 {
		return updateBan(ctx, ban)
	}
	ban.CreatedOn = config.Now()

	existing, e := getBanBySteamID(ctx, ban.SteamID, false)
	if e != nil && !errors.Is(e, errNoResult) {
		return errors.Wrapf(err, "Failed to check existing ban state")
	}
	if ban.BanType <= existing.Ban.BanType {
		return errDuplicate
	}
	return insertBan(ctx, ban)
}

func insertBan(ctx context.Context, ban *model.Ban) error {
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
	err := db.QueryRow(ctx, q, a...).Scan(&ban.BanID)
	if err != nil {
		return dbErr(err)
	}
	return nil
}

func updateBan(ctx context.Context, ban *model.Ban) error {
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
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func dropPerson(ctx context.Context, steamID steamid.SID64) error {
	q, a, e := sb.Delete("person").Where(sq.Eq{"steam_id": steamID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

// SavePerson will insert or update the person record
func SavePerson(ctx context.Context, person *model.Person) error {
	person.UpdatedOn = config.Now()
	if !person.IsNew {
		return updatePerson(ctx, person)
	}
	person.CreatedOn = person.UpdatedOn
	return insertPerson(ctx, person)
}

func updatePerson(ctx context.Context, p *model.Person) error {
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
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func insertPerson(ctx context.Context, p *model.Person) error {
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
	_, err := db.Exec(ctx, q, a...)
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

// getPersonBySteamID returns a person by their steam_id. errNoResult is returned if the steam_id
// is not known.
func getPersonBySteamID(ctx context.Context, sid steamid.SID64) (*model.Person, error) {
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

	p := model.NewPerson(0)
	p.IsNew = false
	p.PlayerSummary = &extra.PlayerSummary{}
	err := db.QueryRow(ctx, q, sid.Int64()).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.IPAddr, &p.CommunityBanned,
		&p.VACBans, &p.GameBans, &p.EconomyBan, &p.DaysSinceLastBan)
	if err != nil {
		return nil, dbErr(err)
	}
	return p, nil
}

func getPeople(ctx context.Context, qf *queryFilter) ([]*model.Person, error) {
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
	rows, err := db.Query(ctx, q, a...)
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
		people = append(people, p)
	}
	return people, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func GetOrCreatePersonBySteamID(ctx context.Context, sid steamid.SID64) (*model.Person, error) {
	p, err := getPersonBySteamID(ctx, sid)
	if err != nil && dbErr(err) == errNoResult {
		p = model.NewPerson(sid)
		if err := SavePerson(ctx, p); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return p, nil
}

// getPersonByDiscordID returns a person by their discord_id
func getPersonByDiscordID(ctx context.Context, did string) (*model.Person, error) {
	q, a, e := sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": did}).
		ToSql()
	if e != nil {
		return nil, e
	}
	p := model.NewPerson(0)
	p.IsNew = false
	p.PlayerSummary = &extra.PlayerSummary{}
	err := db.QueryRow(ctx, q, a...).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans,
		&p.EconomyBan, &p.DaysSinceLastBan)
	if err != nil {
		return nil, dbErr(err)
	}
	return p, nil
}

// GetBanNet returns the BanNet matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func getBanNet(ctx context.Context, ip net.IP) ([]model.BanNet, error) {
	q, _, e := sb.Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From("ban_net").
		Suffix("WHERE $1 <<= cidr").
		ToSql()
	if e != nil {
		return nil, e
	}
	var nets []model.BanNet
	rows, err := db.Query(ctx, q, ip.String())
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

func updateBanNet(ctx context.Context, banNet *model.BanNet) error {
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
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return err
	}
	return nil
}

func insertBanNet(ctx context.Context, banNet *model.BanNet) error {
	q, a, e := sb.Insert("ban_net").
		Columns("cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		Values(banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ValidUntil).
		Suffix("RETURNING net_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.QueryRow(ctx, q, a...).Scan(&banNet.NetID)
	if err != nil {
		return err
	}
	return nil
}

func saveBanNet(ctx context.Context, banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return updateBanNet(ctx, banNet)
	}
	return insertBanNet(ctx, banNet)
}

func dropNetBan(ctx context.Context, ban model.BanNet) error {
	q, a, e := sb.Delete("ban_net").Where(sq.Eq{"net_id": ban.NetID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

func getExpiredBans(ctx context.Context) ([]*model.Ban, error) {
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
	rows, err := db.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.ValidUntil, &b.Source, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, &b)
	}
	return bans, nil
}

//func GetBansTotal(o *queryFilter) (int, error) {
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

// getBans returns all bans that fit the filter criteria passed in
func getBans(ctx context.Context, o *queryFilter) ([]*model.BannedPerson, error) {
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
	rows, err := db.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		b := model.NewBannedPerson()
		if err := rows.Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn, &b.Person.IPAddr,
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

func getBansOlderThan(ctx context.Context, o *queryFilter, t time.Time) ([]model.Ban, error) {
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
	rows, err := db.Query(ctx, q, a...)
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

func getExpiredNetBans(ctx context.Context) ([]model.BanNet, error) {
	q, a, e := sb.
		Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From(string(tableBanNet)).
		Where(sq.Lt{"valid_until": config.Now()}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []model.BanNet
	rows, err := db.Query(ctx, q, a...)
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

func insertFilter(ctx context.Context, rx string) (*model.Filter, error) {
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
	if err := db.QueryRow(ctx, q, a...).Scan(&filter.WordID); err != nil {
		return nil, dbErr(err)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return filter, nil
}

func dropFilter(ctx context.Context, filter *model.Filter) error {
	q, a, e := sb.Delete(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": filter.WordID}).
		ToSql()
	if e != nil {
		return dbErr(e)
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	log.Debugf("Deleted filter: %d", filter.WordID)
	return nil
}

func getFilterByID(ctx context.Context, wordId int) (*model.Filter, error) {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": wordId}).
		ToSql()
	if e != nil {
		return nil, dbErr(e)
	}
	var f model.Filter
	var w string
	if err := db.QueryRow(ctx, q, a...).Scan(&f.WordID, &w, &f.CreatedOn); err != nil {
		return nil, errors.Wrapf(err, "Failed to load filter")
	}
	rx, er := regexp.Compile(w)
	if er != nil {
		return nil, er
	}
	f.Word = rx
	return &f, nil
}

func getFilters(ctx context.Context) ([]*model.Filter, error) {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).ToSql()
	if e != nil {
		return nil, dbErr(e)
	}
	rows, err := db.Query(ctx, q, a...)
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

func insertLog(ctx context.Context, l *model.ServerLog) error {
	q, a, e := sb.Insert(string(tableServerLog)).
		Columns("server_id", "event_type", "payload", "source_id", "target_id", "Created_on").
		Values(l.ServerID, l.EventType, l.Payload, l.SourceID, l.TargetID, l.CreatedOn).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	return nil
}

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func InsertBlockListData(ctx context.Context, d *ip2location.BlockListData) error {
	if len(d.Proxies) > 0 {
		if err := loadProxies(ctx, d.Proxies, false); err != nil {
			return err
		}
	}
	if len(d.Locations4) > 0 {
		if err := loadLocation(ctx, d.Locations4, false); err != nil {
			return err
		}
	}
	if len(d.ASN4) > 0 {
		if err := loadASN(ctx, d.ASN4); err != nil {
			return err
		}
	}
	return nil
}

func getASNRecord(ctx context.Context, ip net.IP) (*ip2location.ASNRecord, error) {
	q, _, e := sb.Select("ip_from", "ip_to", "cidr", "as_num", "as_name").
		From("net_asn").
		Where("$1 << cidr").
		Limit(1).
		ToSql()
	if e != nil {
		return nil, e
	}
	var r ip2location.ASNRecord
	if err := db.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.CIDR, &r.ASNum, &r.ASName); err != nil {
		return nil, err
	}
	return &r, nil
}

func getLocationRecord(ctx context.Context, ip net.IP) (*ip2location.LocationRecord, error) {
	const q = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE $1 <@ ip_range`
	var r ip2location.LocationRecord
	if err := db.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.LatLong.Latitude, &r.LatLong.Longitude); err != nil {
		return nil, err
	}
	return &r, nil
}

func getProxyRecord(ctx context.Context, ip net.IP) (*ip2location.ProxyRecord, error) {
	const q = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1 <@ ip_range`
	var r ip2location.ProxyRecord
	if err := db.QueryRow(ctx, q, ip).
		Scan(&r.IPFrom, &r.IPTo, &r.ProxyType, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.ISP,
			&r.Domain, &r.UsageType, &r.ASN, &r.AS, &r.LastSeen, &r.Threat); err != nil {
		return nil, dbErr(err)
	}
	return &r, nil
}

// ipRecord holds a composite result of ip2location results
type ipRecord struct {
	IP          net.IP
	CreatedOn   time.Time
	CityName    string
	CountryName string
	CountryCode string
	ASName      string
	ASNum       int
	ISP         string
	UsageType   string
	Threat      string
	DomainUsed  string
}

func getPersonIPHistory(ctx context.Context, sid steamid.SID64) ([]ipRecord, error) {
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
	rows, err := db.Query(ctx, q, sid.Int64())
	if err != nil {
		return nil, err
	}
	var records []ipRecord
	defer rows.Close()
	for rows.Next() {
		var r ipRecord
		if errR := rows.Scan(&r.IP, &r.CreatedOn, &r.CityName, &r.CountryName, &r.CountryCode, &r.ASName,
			&r.ASNum, &r.ISP, &r.UsageType, &r.Threat, &r.DomainUsed); errR != nil {
			return nil, errR
		}
		records = append(records, r)
	}
	return records, nil
}

func truncateTable(ctx context.Context, table tableName) error {
	if _, err := db.Exec(ctx, fmt.Sprintf("TRUNCATE %s;", table)); err != nil {
		return dbErr(err)
	}
	return nil
}

func loadASN(ctx context.Context, records []ip2location.ASNRecord) error {
	t0 := time.Now()
	if err := truncateTable(ctx, tableNetASN); err != nil {
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
				r := db.SendBatch(c, &b)
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

func loadLocation(ctx context.Context, records []ip2location.LocationRecord, _ bool) error {
	t0 := time.Now()
	if err := truncateTable(ctx, tableNetLocation); err != nil {
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
				r := db.SendBatch(c, &b)
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

func loadProxies(ctx context.Context, records []ip2location.ProxyRecord, _ bool) error {
	t0 := time.Now()
	if err := truncateTable(ctx, tableNetProxy); err != nil {
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
				r := db.SendBatch(c, &b)
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

func getStats(ctx context.Context) (model.Stats, error) {
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
	var stats model.Stats
	if err := db.QueryRow(ctx, q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); err != nil {
		log.Errorf("Failed to fetch stats: %v", err)
		return model.Stats{}, dbErr(err)
	}
	return stats, nil

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
			return errDuplicate
		default:
			log.Errorf("Unhandled store error: (%s) %s", pgErr.Code, pgErr.Message)
			return err
		}
	}
	if err.Error() == "no rows in result set" {
		return errNoResult
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
func Migrate(action MigrationAction) error {
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
func Import(ctx context.Context, root string) error {
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
				b1, e1 := GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.SteamID))
				if e1 != nil {
					return e1
				}
				b2, e2 := GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.AuthorID))
				if e2 != nil {
					return e2
				}
				sum, err3 := extra.PlayerSummaries(context.Background(), []steamid.SID64{b1.SteamID, b2.SteamID})
				if err3 != nil {
					log.Errorf("Failed to get player summary: %v", err3)
					return err3
				}
				if len(sum) > 0 {
					b1.PlayerSummary = &sum[0]
					if err4 := SavePerson(ctx, b1); err4 != nil {
						return err4
					}
					if b2.SteamID.Valid() && len(sum) > 1 {
						b2.PlayerSummary = &sum[1]
						if err5 := SavePerson(ctx, b2); err5 != nil {
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

				if err4 := saveBan(ctx, bn); err4 != nil {
					return err4
				}
			}
		}
		return nil
	})
}
