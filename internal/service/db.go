package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/extra"
	"io/fs"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	db           *pgxpool.Pool
	errNoResult  = errors.New("No results found")
	errDuplicate = errors.New("Duplicate entity")

	// Use $ for pg based queries
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
)

type QueryOpts struct {
	Limit     uint64
	Offset    uint64
	OrderDesc bool
	OrderBy   string
}

func (o QueryOpts) order() string {
	if o.OrderDesc {
		return "DESC"
	}
	return "ASC"
}

func newQueryOpts() QueryOpts {
	return QueryOpts{
		Limit:     100,
		Offset:    0,
		OrderDesc: false,
		OrderBy:   "",
	}
}

func newSearchQueryOpts(query string) searchQueryOpts {
	o := newQueryOpts()
	return searchQueryOpts{
		query,
		o,
	}
}

// Init sets up underlying required services.
func Init(dsn string) {
	dbConn, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	db = dbConn
}

func Close() {
	db.Close()
}

func tokenValid(token string) bool {
	if len(token) != 40 {
		return false
	}
	var s int
	q, a, e := sb.Select("server_id").From("server").Where(sq.Eq{"token": token}).ToSql()
	if e != nil {
		log.Errorf("Failed to select token: %v", e)
		return false
	}
	if err := db.QueryRow(context.Background(), q, a...).
		Scan(&s); err != nil {
		return false
	}
	return s > 0
}

func getServer(serverID int64) (model.Server, error) {
	var s model.Server
	q, a, e := sb.Select("server_id", "short_name", "token", "address", "port", "rcon",
		"token_created_on", "created_on", "updated_on", "reserved_slots").
		From("server").
		Where(sq.Eq{"server_id": serverID}).
		ToSql()
	if e != nil {
		return model.Server{}, e
	}
	if err := db.QueryRow(context.Background(), q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

func getServers() ([]model.Server, error) {
	var servers []model.Server
	q, _, e := sb.Select("server_id", "short_name", "token", "address", "port", "rcon",
		"token_created_on", "created_on", "updated_on", "reserved_slots").
		From("server").
		ToSql()
	if e != nil {
		return nil, e
	}
	rows, err := db.Query(context.Background(), q)
	if err != nil {
		return []model.Server{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.Server
		if err := rows.Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return servers, nil
}

func getServerByName(serverName string) (model.Server, error) {
	var s model.Server
	q, a, e := sb.Select("server_id", "short_name", "token", "address", "port", "rcon",
		"token_created_on", "created_on", "updated_on", "reserved_slots").
		From("server").
		Where(sq.Eq{"short_name": serverName}).
		ToSql()
	if e != nil {
		return model.Server{}, e
	}
	if err := db.QueryRow(context.Background(), q, a...).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

// SaveServer updates or creates the server data in the database
func SaveServer(server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return updateServer(server)
	}
	server.CreatedOn = config.Now()
	return insertServer(server)
}

func insertServer(s *model.Server) error {
	q, a, e := sb.Insert("server").
		Columns("short_name", "token", "address", "port",
			"rcon", "token_created_on", "created_on", "updated_on", "password", "reserved_slots").
		Values(s.ServerName, s.Token, s.Address, s.Port, s.RCON, s.TokenCreatedOn,
			s.CreatedOn, s.UpdatedOn, s.Password, s.ReservedSlots).
		Suffix("RETURNING server_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.QueryRow(context.Background(), q, a...).Scan(&s.ServerID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func updateServer(s *model.Server) error {
	s.UpdatedOn = config.Now()
	q, a, e := sb.Update("server").
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
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update s")
	}
	return nil
}

func dropServer(serverID int64) error {
	q, a, e := sb.Delete("server").Where(sq.Eq{"server_id": serverID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return err
	}
	return nil
}

func dropBan(ban *model.Ban) error {
	q, a, e := sb.Delete("ban").Where(sq.Eq{"ban_id": ban.BanID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func getBanByColumn(column string, identifier interface{}, full bool) (*model.BannedPerson, error) {
	q, a, e := sb.Select(
		"b.ban_id", "b.steam_id", "b.author_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.ban_source", "b.valid_until", "b.created_on", "b.updated_on",

		"p.steam_id as sid2", "p.created_on as created_on2", "p.updated_on as updated_on2", "p.ip_addr", "p.communityvisibilitystate", "p.profilestate",
		"p.personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull", "p.avatarhash",
		"p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode", "p.loccityid").
		From("ban b").
		LeftJoin("person p ON b.steam_id = p.steam_id").
		GroupBy("b.ban_id, p.steam_id").
		Where(sq.Eq{fmt.Sprintf("b.%s", column): identifier}).
		OrderBy("b.created_on DESC").
		Limit(1).
		ToSql()
	if e != nil {
		return nil, e
	}
	b := model.NewBannedPerson()

	if err := db.QueryRow(context.Background(), q, a...).
		Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn, &b.Person.IPAddr,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName,
			&b.Person.ProfileURL, &b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull,
			&b.Person.AvatarHash, &b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID); err != nil {
		return nil, DBErr(err)
	}
	if full {
		b.HistoryChat = getChatHistory(b.Person.SteamID)
		b.HistoryConnections = []string{}
		b.HistoryIP = getIPHistory(b.Person.SteamID)
		b.HistoryPersonaName = []string{}
	}
	return b, nil
}

func getBanBySteamID(steamID steamid.SID64, full bool) (*model.BannedPerson, error) {
	return getBanByColumn("steam_id", steamID, full)
}

func getBanByBanID(banID uint64, full bool) (*model.BannedPerson, error) {
	return getBanByColumn("ban_id", banID, full)
}

func getChatHistory(sid64 steamid.SID64) []string {
	return nil
}

func getIPHistory(sid64 steamid.SID64) []net.IP {
	return []net.IP{net.ParseIP("127.0.0.1")}
}

func getAppeal(banID int) (model.Appeal, error) {
	q, a, e := sb.Select("appeal_id", "ban_id", "appeal_text", "appeal_state",
		"email", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.Eq{"ban_id": banID}).
		ToSql()
	if e != nil {
		return model.Appeal{}, e
	}
	var ap model.Appeal
	if err := db.QueryRow(context.Background(), q, a...).
		Scan(&ap.AppealID, &ap.BanID, &ap.AppealText, &ap.AppealState, &ap.Email, &ap.CreatedOn,
			&ap.UpdatedOn); err != nil {
		return model.Appeal{}, err
	}
	return ap, nil
}

func updateAppeal(appeal *model.Appeal) error {
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
	_, err := db.Exec(context.Background(), q, a...)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func insertAppeal(ap *model.Appeal) error {
	q, a, e := sb.Insert("ban_appeal").
		Columns("ban_id", "appeal_text", "appeal_state", "email", "created_on", "updated_on").
		Values(ap.BanID, ap.AppealText, ap.AppealState, ap.Email, ap.CreatedOn, ap.UpdatedOn).
		Suffix("RETURNING appeal_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.QueryRow(context.Background(), q, a...).Scan(&ap.AppealID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func saveAppeal(appeal *model.Appeal) error {
	appeal.UpdatedOn = config.Now()
	if appeal.AppealID > 0 {
		return updateAppeal(appeal)
	}
	appeal.CreatedOn = config.Now()
	return insertAppeal(appeal)
}

func SaveBan(ban *model.Ban) error {
	ban.UpdatedOn = config.Now()
	if ban.BanID > 0 {
		return updateBan(ban)
	}
	ban.CreatedOn = config.Now()
	return insertBan(ban)
}

func insertBan(ban *model.Ban) error {
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
	err := db.QueryRow(context.Background(), q, a...).Scan(&ban.BanID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func updateBan(ban *model.Ban) error {
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
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func SavePerson(person *model.Person) error {
	person.UpdatedOn = config.Now()
	if !person.IsNew {
		return updatePerson(person)
	}
	person.CreatedOn = person.UpdatedOn
	return insertPerson(person)
}

func updatePerson(p *model.Person) error {
	p.UpdatedOn = config.Now()
	q, a, e := sb.Update("person").
		Set("updated_on", p.UpdatedOn).
		Set("ip_addr", p.IPAddr).
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
		Where(sq.Eq{"steam_id": p.SteamID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func insertPerson(p *model.Person) error {
	q, a, e := sb.
		Insert("person").
		Columns(
			"created_on", "updated_on", "steam_id", "ip_addr", "communityvisibilitystate",
			"profilestate", "personaname", "profileurl", "avatar", "avatarmedium", "avatarfull",
			"avatarhash", "personastate", "realname", "timecreated", "loccountrycode", "locstatecode", "loccityid").
		Values(p.CreatedOn, p.UpdatedOn, p.SteamID, p.IPAddr,
			p.CommunityVisibilityState, p.ProfileState, p.PersonaName, p.ProfileURL,
			p.Avatar, p.AvatarMedium, p.AvatarFull, p.AvatarHash, p.PersonaState, p.RealName, p.TimeCreated,
			p.LocCountryCode, p.LocStateCode, p.LocCityID).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.Exec(context.Background(), q, a...)
	if err != nil {
		return DBErr(err)
	}
	p.IsNew = false
	return nil
}

// getPersonBySteamID returns a person by their steam_id. errNoResult is returned if the steam_id
// is not known.
func getPersonBySteamID(sid steamid.SID64) (*model.Person, error) {
	q, a, e := sb.Select("steam_id", "created_on", "updated_on", "ip_addr",
		"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
		"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
		"loccountrycode", "locstatecode", "loccityid").
		From("person").
		Where(sq.Eq{"steam_id": sid}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var p model.Person
	p.PlayerSummary = &extra.PlayerSummary{}
	err := db.QueryRow(context.Background(), q, a...).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.IPAddr, &p.CommunityVisibilityState,
		&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
		&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID)
	if err != nil {
		return nil, DBErr(err)
	}
	return &p, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func GetOrCreatePersonBySteamID(sid steamid.SID64) (*model.Person, error) {
	p, err := getPersonBySteamID(sid)
	if err != nil && DBErr(err) == errNoResult {
		p = model.NewPerson(sid)
		if err := SavePerson(p); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return p, nil
}

// GetBanNet returns the BanNet matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func getBanNet(ip net.IP) ([]model.BanNet, error) {
	q, _, e := sb.Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From("ban_net").
		Suffix("WHERE $1 <<= cidr").
		ToSql()
	if e != nil {
		return nil, e
	}
	var nets []model.BanNet
	rows, err := db.Query(context.Background(), q, ip.String())
	if err != nil {
		return nil, DBErr(err)
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

func updateBanNet(banNet *model.BanNet) error {
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
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return err
	}
	return nil
}

func insertBanNet(banNet *model.BanNet) error {
	q, a, e := sb.Insert("ban_net").
		Columns("cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		Values(banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ValidUntil).
		Suffix("RETURNING net_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.QueryRow(context.Background(), q, a...).Scan(&banNet.NetID)
	if err != nil {
		return err
	}
	return nil
}

func saveBanNet(banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return updateBanNet(banNet)
	}
	return insertBanNet(banNet)
}

func dropNetBan(ban model.BanNet) error {
	q, a, e := sb.Delete("ban_net").Where(sq.Eq{"net_id": ban.NetID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func getExpiredBans() ([]*model.Ban, error) {
	q, a, e := sb.Select(
		"ban_id", "steam_id", "author_id", "ban_type", "reason", "reason_text", "note",
		"valid_until", "ban_source", "created_on", "updated_on").
		From("ban").
		Where(sq.Lt{"valid_until": config.Now()}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []*model.Ban
	rows, err := db.Query(context.Background(), q, a...)
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

type searchQueryOpts struct {
	SearchTerm string
	QueryOpts
}

func GetBansTotal(o searchQueryOpts) (int, error) {
	q, _, e := sb.Select("count(*) as total_rows").From("ban").ToSql()
	if e != nil {
		return 0, e
	}
	var total int
	if err := db.QueryRow(context.Background(), q).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func GetBans(o searchQueryOpts) ([]*model.BannedPerson, error) {
	q, a, e := sb.Select(
		"b.ban_id", "b.steam_id", "b.author_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.ban_source", "b.valid_until", "b.created_on", "b.updated_on",
		"p.steam_id", "p.created_on", "p.updated_on", "p.ip_addr", "p.communityvisibilitystate", "p.profilestate",
		"p.personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull", "p.avatarhash",
		"p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode", "p.loccityid").
		From("ban b").
		LeftJoin("person p on p.steam_id = b.steam_id").
		OrderBy(fmt.Sprintf("b.%s", o.OrderBy)).
		Limit(o.Limit).
		Offset(o.Offset).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []*model.BannedPerson
	rows, err := db.Query(context.Background(), q, a...)
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
			&b.Person.LocStateCode, &b.Person.LocCityID); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func getBansOlderThan(o QueryOpts, t time.Time) ([]model.Ban, error) {
	q, a, e := sb.
		Select("ban_id", "steam_id", "author_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "created_on", "updated_on", "ban_source").
		From("ban").
		Where(sq.Lt{"updated_on": t}).
		Limit(o.Limit).Offset(o.Offset).ToSql()
	if e != nil {
		return nil, e
	}
	var bans []model.Ban
	rows, err := db.Query(context.Background(), q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.ValidUntil, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func getExpiredNetBans() ([]model.BanNet, error) {
	q, a, e := sb.
		Select("net_id", "cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		From("ban_net").
		Where(sq.Lt{"valid_until": config.Now()}).
		ToSql()
	if e != nil {
		return nil, e
	}
	var bans []model.BanNet
	rows, err := db.Query(context.Background(), q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BanNet
		if err := rows.Scan(&b.NetID, &b.CIDR, &b.Source, &b.CreatedOn, &b.UpdatedOn, &b.Reason, &b.ValidUntil); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func GetFilteredWords() ([]string, error) {
	q, a, e := sb.Select("word").From("filtered_word").ToSql()
	if e != nil {
		return nil, e
	}
	var words []string
	rows, err := db.Query(context.Background(), q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, nil
}

func SaveFilteredWord(word string) error {
	q, a, e := sb.Insert("filtered_word").Columns("word").Values(word).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func InsertLog(l *model.ServerLog) error {
	q, a, e := sb.Insert("server_log").
		Columns("server_id", "event_type", "payload", "source_id", "target_id", "Created_on").
		Values(l.ServerID, l.EventType, l.Payload, l.SourceID, l.TargetID, l.CreatedOn).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.Exec(context.Background(), q, a...); err != nil {
		return DBErr(err)
	}
	return nil
}

func getStats() (model.Stats, error) {
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
	if err := db.QueryRow(context.Background(), q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); err != nil {
		log.Errorf("Failed to fetch stats: %v", err)
		return model.Stats{}, DBErr(err)
	}
	return stats, nil

}

func DBErr(err error) error {
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

//go:embed "schema.sql"
var schema string

func Migrate(recreate bool) error {
	const q = `DROP TABLE IF EXISTS %s;`
	if recreate {
		for _, t := range []string{"ban_appeal", "filtered_word", "ban_net", "ban", "person_names", "person"} {
			_, err := db.Exec(context.Background(), fmt.Sprintf(q, t))
			if err != nil {
				return errors.Wrap(err, "Could not remove all tables")
			}
		}
	}
	_, err := db.Exec(context.Background(), schema)
	if err != nil {
		return errors.Wrap(err, "Could not create new schema")
	}
	_, err = GetOrCreatePersonBySteamID(config.General.Owner)
	if err != nil {
		log.Fatalf("Error loading system user: %v", err)
	}
	return nil
}

func Import(root string) error {
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
			log.Infoln(imported)
			for _, im := range imported {
				b1, e1 := GetOrCreatePersonBySteamID(steamid.SID64(im.SteamID))
				if e1 != nil {
					return e1
				}
				b2, e2 := GetOrCreatePersonBySteamID(steamid.SID64(im.AuthorID))
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
					if err4 := SavePerson(b1); err4 != nil {
						return err4
					}
					if b2.SteamID.Valid() && len(sum) > 1 {
						b2.PlayerSummary = &sum[1]
						if err5 := SavePerson(b2); err5 != nil {
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

				if err3 := SaveBan(bn); err3 != nil {
					return err3
				}
			}
		}
		return nil
	})
}
