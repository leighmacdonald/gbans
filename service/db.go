package service

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"time"

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
	ErrNoResult  = errors.New("No results found")
	ErrDuplicate = errors.New("Duplicate entity")
)

type QueryOpts struct {
	Limit     int
	Offset    int
	OrderDesc bool
	OrderBy   string
}

func (o QueryOpts) Order() string {
	if o.OrderDesc {
		return "DESC"
	}
	return "ASC"
}

func NewQueryOpts() QueryOpts {
	return QueryOpts{
		Limit:     100,
		Offset:    0,
		OrderDesc: false,
		OrderBy:   "",
	}
}

func NewSearchQueryOpts(query string) SearchQueryOpts {
	o := NewQueryOpts()
	return SearchQueryOpts{
		query,
		o,
	}
}

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

// Probably shouldn't be here
func TokenValid(token string) bool {
	if len(token) != 40 {
		return false
	}
	var s int
	const q = `
		SELECT server_id FROM server WHERE token = $1`
	if err := db.QueryRow(context.Background(), q, token).
		Scan(&s); err != nil {
		return false
	}
	return s > 0
}

func GetServer(serverID int64) (model.Server, error) {
	var s model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server
		WHERE server_id = $1`
	if err := db.QueryRow(context.Background(), q, serverID).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

func GetServers() ([]model.Server, error) {
	var servers []model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server`
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

func GetServerByName(serverName string) (model.Server, error) {
	var s model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server
		WHERE short_name = $1`
	if err := db.QueryRow(context.Background(), q, serverName).
		Scan(&s.ServerID, &s.ServerName, &s.Token, &s.Address, &s.Port,
			&s.RCON, &s.TokenCreatedOn, &s.CreatedOn, &s.UpdatedOn, &s.ReservedSlots); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

func SaveServer(server *model.Server) error {
	server.UpdatedOn = config.Now()
	if server.ServerID > 0 {
		return updateServer(server)
	}
	server.CreatedOn = config.Now()
	return insertServer(server)
}

func insertServer(s *model.Server) error {
	const q = `
		INSERT INTO server (
		    short_name, token, address, port, rcon, token_created_on, created_on, updated_on, password, reserved_slots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING server_id`

	err := db.QueryRow(context.Background(), q, s.ServerName, s.Token, s.Address, s.Port, s.RCON, s.TokenCreatedOn,
		s.CreatedOn, s.UpdatedOn, s.Password, s.ReservedSlots).Scan(&s.ServerID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func updateServer(s *model.Server) error {
	const q = `
		UPDATE server
		SET short_name = $1, token = $2, address = $3, port = $4,
		    rcon = $5, token_created_on = $6, updated_on = $7,
		    reserved_slots = $8
		WHERE server_id = $9`
	s.UpdatedOn = config.Now()
	if _, err := db.Exec(context.Background(), q, s.ServerName, s.Token, s.Address, s.Port, s.RCON,
		s.TokenCreatedOn, s.UpdatedOn, s.ReservedSlots, s.ServerID); err != nil {
		return errors.Wrapf(err, "Failed to update s")
	}
	return nil
}

func DropServer(serverID int64) error {
	const q = `DELETE FROM server WHERE server_id = $1`
	if _, err := db.Exec(context.Background(), q, serverID); err != nil {
		return err
	}
	return nil
}

func DropBan(ban model.Ban) error {
	const q = `DELETE FROM ban WHERE ban_id = $1`
	if _, err := db.Exec(context.Background(), q, ban); err != nil {
		return DBErr(err)
	}
	return nil
}

func GetBan(steamID steamid.SID64) (model.Ban, error) {
	const q = `
		SELECT 
			ban_id, steam_id, ban_type, reason, note, until,
			created_on, updated_on, reason_text, ban_source
		FROM ban
		WHERE ($1 > 0 AND steam_id = $1)`
	var b model.Ban
	if err := db.QueryRow(context.Background(), q, steamID.Int64()).
		Scan(&b.BanID, &b.SteamID, &b.BanType, &b.Reason, &b.Note, &b.Until, &b.CreatedOn,
			&b.UpdatedOn, &b.ReasonText, &b.Source); err != nil {
		return model.Ban{}, DBErr(err)
	}
	return b, nil
}

func GetAppeal(banID int) (model.Appeal, error) {
	const q = `SELECT appeal_id, ban_id, appeal_text, appeal_state, 
       email, created_on, updated_on FROM ban_appeal a
       WHERE a.ban_id = $1`
	var a model.Appeal
	if err := db.QueryRow(context.Background(), q, banID).
		Scan(&a.AppealID, &a.BanID, &a.AppealText, &a.AppealState, &a.Email, &a.CreatedOn,
			&a.UpdatedOn); err != nil {
		return model.Appeal{}, err
	}
	return a, nil
}

func updateAppeal(appeal *model.Appeal) error {
	const q = `UPDATE ban_appeal SET appeal_text = $1, appeal_state = $2, email = $3,
		updated_on = $4 WHERE appeal_id = $5`
	_, err := db.Exec(context.Background(), q, appeal.AppealText, appeal.AppealState, appeal.Email,
		appeal.UpdatedOn, appeal.AppealID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func insertAppeal(a *model.Appeal) error {
	const q = `INSERT INTO ban_appeal (ban_id, appeal_text, appeal_state, email, created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING appeal_id`
	err := db.QueryRow(context.Background(), q, a.BanID, a.AppealText, a.AppealState, a.Email, a.CreatedOn,
		a.UpdatedOn).Scan(&a.AppealID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func SaveAppeal(appeal *model.Appeal) error {
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
	const q = `
		INSERT INTO ban (
			steam_id, author_id, ban_type, reason, reason_text, 
			note, until, created_on, updated_on, ban_source) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING ban_id`
	err := db.QueryRow(context.Background(), q, ban.SteamID, ban.AuthorID, ban.BanType, ban.Reason, ban.ReasonText,
		ban.Note, ban.Until, ban.CreatedOn, ban.UpdatedOn, ban.Source).Scan(&ban.BanID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func updateBan(ban *model.Ban) error {
	const q = `
		UPDATE ban 
		SET ban_type = $2, reason = $3, reason_text = $4, 
			note = $5, updated_on = $6, ban_source = $7
		WHERE ban_id = $1`
	if _, err := db.Exec(context.Background(), q,
		ban.BanID, ban.BanType, ban.Reason, ban.ReasonText, ban.Note, ban.UpdatedOn, ban.Source); err != nil {
		return DBErr(err)
	}
	return nil
}

func SavePerson(person *model.Person) error {
	person.UpdatedOn = config.Now()
	if person.CreatedOn.UTC().Unix() > 0 {
		return updatePerson(person)
	}
	person.CreatedOn = person.UpdatedOn
	return insertPerson(person)
}

func updatePerson(p *model.Person) error {
	const q = `
		UPDATE person
		SET updated_on = $1, steam_id = $2, ip_addr = $3, communityvisibilitystate = $4, 
			profilestate = $5, personaname = $6, profileurl = $7, avatar = $8, avatarmedium = $9, avatarfull = $10, 
			avatarhash = $11, personastate = $12, realname = $13, timecreated = $14, loccountrycode = $15,
			locstatecode = $16, loccityid = $17
		WHERE steam_id = $18`
	p.UpdatedOn = config.Now()
	if _, err := db.Exec(context.Background(), q, p.UpdatedOn, p.SteamID, p.IPAddr,
		p.CommunityVisibilityState, p.ProfileState, p.PersonaName, p.ProfileURL,
		p.Avatar, p.AvatarMedium, p.AvatarFull, p.AvatarHash, p.PersonaState, p.RealName, p.TimeCreated,
		p.LocCountryCode, p.LocStateCode, p.LocCityID, p.SteamID); err != nil {
		return DBErr(err)
	}
	return nil
}

func insertPerson(p *model.Person) error {
	const q = `
		INSERT INTO person (
			created_on, updated_on, steam_id, ip_addr, communityvisibilitystate, profilestate, personaname,
			profileurl, avatar, avatarmedium, avatarfull, avatarhash, personastate, realname, timecreated, loccountrycode,
			locstatecode, loccityid
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	_, err := db.Exec(context.Background(), q, p.CreatedOn, p.UpdatedOn, p.SteamID, p.IPAddr,
		p.CommunityVisibilityState, p.ProfileState, p.PersonaName, p.ProfileURL,
		p.Avatar, p.AvatarMedium, p.AvatarFull, p.AvatarHash, p.PersonaState, p.RealName, p.TimeCreated,
		p.LocCountryCode, p.LocStateCode, p.LocCityID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func GetPersonBySteamID(sid steamid.SID64) (model.Person, error) {
	const q = `
		SELECT 
		    steam_id, created_on, updated_on, ip_addr, communityvisibilitystate, profilestate, 
       		personaname, profileurl, avatar, avatarmedium, avatarfull, avatarhash, personastate, realname, 
       		timecreated, loccountrycode, locstatecode, loccityid 
		FROM person 
		WHERE steam_id = $1`
	var p model.Person
	err := db.QueryRow(context.Background(), q, sid).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.IPAddr, &p.CommunityVisibilityState,
		&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
		&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID)
	if err != nil && DBErr(err) == ErrNoResult {
		p.SteamID = sid
		if err := SavePerson(&p); err != nil {
			return model.Person{}, err
		}
	} else if err != nil {
		return model.Person{}, err
	}
	return p, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func GetOrCreatePersonBySteamID(sid steamid.SID64) (model.Person, error) {
	p, err := GetPersonBySteamID(sid)
	if err != nil && DBErr(err) == ErrNoResult {
		p.SteamID = sid
		if err := SavePerson(&p); err != nil {
			return model.Person{}, err
		}
	} else if err != nil {
		return model.Person{}, err
	}
	return p, nil
}

// GetBanNet returns the BanNet matching intersecting the supplied ip
// TODO keep nets in memory?
func GetBanNet(ip string) (model.BanNet, error) {
	addr := net.ParseIP(ip)
	const q = `SELECT net_id, cidr::inet, source, created_on, updated_on, reason, until FROM ban_net`
	var nets []model.BanNet
	rows, err := db.Query(context.Background(), q)
	if err != nil {
		return model.BanNet{}, DBErr(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n model.BanNet
		if err := rows.Scan(&n.NetID, &n.CIDR, &n.Source, &n.CreatedOn, &n.UpdatedOn, &n.Reason, &n.Until); err != nil {
			return model.BanNet{}, err
		}
		nets = append(nets, n)
	}
	for _, n := range nets {
		_, ipNet, err := net.ParseCIDR(n.CIDR)
		if err != nil {
			continue
		}
		if ipNet.Contains(addr) {
			return n, nil
		}
	}
	return model.BanNet{}, ErrNoResult
}

func updateBanNet(banNet *model.BanNet) error {
	const q = `
		UPDATE ban_net SET cidr = $2, source = $3, updated_on = $4, until = $5
		WHERE net_id = $1`
	if _, err := db.Exec(context.Background(), q,
		banNet.NetID, banNet.CIDR, banNet.Source, banNet.UpdatedOn, banNet.Until); err != nil {
		return err
	}
	return nil
}

func insertBanNet(banNet *model.BanNet) error {
	const q = `
		INSERT INTO ban_net (cidr, source, created_on, updated_on, reason, until) 
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING net_id`
	err := db.QueryRow(context.Background(), q,
		banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.Until).Scan(&banNet.NetID)
	if err != nil {
		return err
	}
	return nil
}

func SaveBanNet(banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return updateBanNet(banNet)
	}
	return insertBanNet(banNet)
}

func DropNetBan(ban model.BanNet) error {
	const q = `DELETE FROM ban_net WHERE net_id = $1`
	if _, err := db.Exec(context.Background(), q, ban.NetID); err != nil {
		return DBErr(err)
	}
	return nil
}

func GetExpiredBans() ([]model.Ban, error) {
	const q = `
		SELECT 
		    b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason, b.reason_text, b.note, b.ban_source,
			b.until, b.created_on, b.updated_on
		FROM ban b 
		WHERE until < $1`
	var bans []model.Ban
	rows, err := db.Query(context.Background(), q, config.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.Until, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

type SearchQueryOpts struct {
	SearchTerm string
	QueryOpts
}

func GetBans(o SearchQueryOpts) ([]model.BannedPerson, error) {
	const q = `
		SELECT 
		    b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason, b.reason_text, b.note, b.ban_source,
			b.until, b.created_on, b.updated_on, p.personaname, p.profileurl, p.avatar, p.avatarmedium
		FROM ban b 
		LEFT OUTER JOIN person p on b.steam_id = p.steam_id
		ORDER BY $1 %s LIMIT $2 OFFSET $3
	`
	q2 := fmt.Sprintf(q, o.Order())
	var bans []model.BannedPerson
	rows, err := db.Query(context.Background(), q2, o.OrderBy, o.Limit, o.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BannedPerson
		if err := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.Until, &b.CreatedOn, &b.UpdatedOn, &b.PersonaName, &b.ProfileURL, &b.Avatar, &b.AvatarMedium,
		); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func GetBansTotal() int {
	var c int
	if err := db.QueryRow(context.Background(), `SELECT count(ban_id) FROM ban`).Scan(&c); err != nil {
		return 0
	}
	return c
}

func GetBansOlderThan(o QueryOpts, t time.Time) ([]model.Ban, error) {
	const q = `SELECT * FROM ban WHERE updated_on < $1 LIMIT $2 OFFSET $3`
	var bans []model.Ban
	rows, err := db.Query(context.Background(), q, t, o.Limit, o.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.Until, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func GetExpiredNetBans() ([]model.BanNet, error) {
	const q = `SELECT net_id, cidr, source, created_on, updated_on, reason, until FROM ban_net WHERE until < $1`
	var bans []model.BanNet
	rows, err := db.Query(context.Background(), q, config.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BanNet
		if err := rows.Scan(&b.NetID, &b.CIDR, &b.Source, &b.CreatedOn, &b.UpdatedOn, &b.Reason, &b.Until); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func GetFilteredWords() ([]string, error) {
	const q = `SELECT word FROM filtered_word`
	var words []string
	rows, err := db.Query(context.Background(), q)
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
	const q = `INSERT INTO filtered_word (word) VALUES ($1)`
	if _, err := db.Exec(context.Background(), q, word); err != nil {
		return DBErr(err)
	}
	return nil
}

func GetStats() (model.Stats, error) {
	const q = `
		SELECT
    (SELECT COUNT(ban_id) FROM ban) as bans_total,
    (SELECT COUNT(ban_id) FROM ban WHERE created_on
         BETWEEN ((julianday('now') - 2440587.5)*86400.0 - 86400) AND (julianday('now') - 2440587.5)*86400.0) as bans_day,
    (SELECT COUNT(ban_id) FROM ban WHERE created_on
         BETWEEN ((julianday('now') - 2440587.5)*86400.0 - (86400 * 24)) AND (julianday('now') - 2440587.5)*86400.0) as bans_month,
    (SELECT COUNT(net_id) FROM ban_net) as ban_cidr,
    (SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 0 ) as appeals_open,
    (SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 1 OR appeal_state = 2 ) as appeals_closed,
    (SELECT COUNT(word_id) FROM filtered_word) as filtered_words,
    (SELECT COUNT(server_id) FROM server) as servers_total
`
	var stats model.Stats
	if err := db.QueryRow(context.Background(), q).Scan(&stats.BansTotal, &stats.BansDay, &stats.BansMonth,
		&stats.BansCIDRTotal, &stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
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
