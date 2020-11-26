package store

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

var (
	db           *sqlx.DB
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

func Init(path string) {
	db = sqlx.MustConnect("sqlite3", path)
	// FIXME
	db.MustExec(schema)
	_, err := GetOrCreatePersonBySteamID(1)
	if err != nil {
		log.Fatalf("Error loading system user: %v", err)
	}
}

func Close() error {
	return db.Close()
}

// Probably shouldn't be here
func TokenValid(token string) bool {
	if len(token) != 40 {
		return false
	}
	var s model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on 
		FROM server
		WHERE token = $1`
	if err := db.Get(&s, q, token); err != nil {
		return false
	}
	return true
}

func GetServer(serverID int64) (model.Server, error) {
	var s model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server
		WHERE server_id = $1`
	if err := db.Get(&s, q, serverID); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

func GetServers() ([]model.Server, error) {
	var s []model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server`
	if err := db.Select(&s, q); err != nil {
		return []model.Server{}, err
	}
	return s, nil
}

func GetServerByName(serverName string) (model.Server, error) {
	var s model.Server
	const q = `
		SELECT 
		    server_id, short_name, token, address, port, rcon,
			token_created_on, created_on, updated_on, reserved_slots
		FROM server
		WHERE short_name = $1`
	if err := db.Get(&s, q, serverName); err != nil {
		return model.Server{}, err
	}
	return s, nil
}

func SaveServer(server *model.Server) error {
	if server.ServerID > 0 {
		return updateServer(server)
	}
	return insertServer(server)
}

func insertServer(server *model.Server) error {
	const q = `
		INSERT INTO server (
		    short_name, token, address, port, rcon, token_created_on, created_on, updated_on, password, reserved_slots) 
		VALUES (:short_name, :token, :address, :port, :rcon, :token_created_on, :created_on, :updated_on, :password, :reserved_slots);`
	server.CreatedOn = time.Now().Unix()
	server.UpdatedOn = time.Now().Unix()
	res, err := db.NamedExec(q, server)
	if err != nil {
		return DBErr(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return errors.Wrapf(err, "Failed to load last inserted ID")
	}
	server.ServerID = i
	return nil
}

func updateServer(server *model.Server) error {
	const q = `
		UPDATE server 
		SET short_name = :short_name, token = :token, address = :address, port = :port,
		    rcon = :rcon, token_created_on = :token_created_on, updated_on = :updated_on,
		    reserved_slots = :reserved_slots
		WHERE server_id = :server_id`
	server.UpdatedOn = time.Now().Unix()
	if _, err := db.NamedExec(q, server); err != nil {
		return errors.Wrapf(err, "Failed to update server")
	}
	return nil
}

func DropServer(serverID int64) error {
	const q = `DELETE FROM server WHERE server_id = $1`
	if _, err := db.Exec(q, serverID); err != nil {
		return err
	}
	return nil
}

func DropBan(ban model.Ban) error {
	const q = `DELETE FROM ban WHERE ban_id = :ban_id`
	if _, err := db.NamedExec(q, ban); err != nil {
		return DBErr(err)
	}
	return nil
}

func GetBan(steamID steamid.SID64) (model.Ban, error) {
	const q = `
		SELECT 
			b.ban_id, b.steam_id, b.ban_type, b.reason, b.note,  b.until,
			b.created_on, b.updated_on, b.reason_text, b.ban_source
		FROM ban b
		WHERE ($1 > 0 AND b.steam_id = $1)`
	var b model.Ban
	if err := db.Get(&b, q, steamID.Int64()); err != nil {
		return model.Ban{}, DBErr(err)
	}
	return b, nil
}

func GetAppeal(banID int) (model.Appeal, error) {
	const q = `SELECT appeal_id, ban_id, appeal_text, appeal_state, 
       email, created_on, updated_on FROM ban_appeal a
       WHERE a.ban_id = $1`
	var a model.Appeal
	if err := db.Get(&a, q, banID); err != nil {
		return model.Appeal{}, err
	}
	return a, nil
}

func updateAppeal(appeal *model.Appeal) error {
	const q = `UPDATE ban_appeal SET appeal_text = :appeal_text, appeal_state = :appeal_state, email = :email,
		updated_on = :updated_on WHERE appeal_id = :appeal_id`
	_, err := db.NamedExec(q, appeal)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func insertAppeal(appeal *model.Appeal) error {
	const q = `INSERT INTO ban_appeal (ban_id, appeal_text, appeal_state, email, created_on, updated_on)
		VALUES (:ban_id, :appeal_text, :appeal_state, :email, :created_on, :updated_on)`
	res, err := db.NamedExec(q, appeal)
	if err != nil {
		return DBErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return DBErr(err)
	}
	appeal.AppealID = int(id)
	return nil
}

func SaveAppeal(appeal *model.Appeal) error {
	appeal.UpdatedOn = time.Now().Unix()
	if appeal.AppealID > 0 {
		return updateAppeal(appeal)
	}
	appeal.CreatedOn = time.Now().Unix()
	return insertAppeal(appeal)
}

func SaveBan(ban *model.Ban) error {
	ban.UpdatedOn = time.Now().Unix()
	if ban.BanID > 0 {
		return updateBan(ban)
	}
	ban.CreatedOn = time.Now().Unix()
	return insertBan(ban)
}

func insertBan(ban *model.Ban) error {
	const q = `
		INSERT INTO ban (
			steam_id, author_id, ban_type, reason, reason_text, 
			note, until, created_on, updated_on, ban_source) 
		VALUES (:steam_id, :author_id,:ban_type, :reason, :reason_text, :note, 
		:until, :created_on, :updated_on, :ban_source)`
	res, err := db.NamedExec(q, ban)
	if err != nil {
		return DBErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return errors.Wrapf(err, "Failed to load last inserted ID")
	}
	ban.BanID = id
	return nil
}

func updateBan(ban *model.Ban) error {
	const q = `
		UPDATE ban 
		SET ban_type = :ban_type, reason = :reason, reason_text = :reason_text, 
			note = :note, updated_on = :updated_on, ban_source = :ban_source
		WHERE ban_id = :ban_id`
	if _, err := db.NamedExec(q, ban); err != nil {
		return DBErr(err)
	}
	return nil
}

func SavePerson(person *model.Person) error {
	person.UpdatedOn = time.Now().Unix()
	if person.CreatedOn > 0 {
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
	p.UpdatedOn = time.Now().Unix()
	if _, err := db.Exec(q, p.UpdatedOn, p.SteamID, p.IPAddr,
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
	_, err := db.Exec(q, p.CreatedOn, p.UpdatedOn, p.SteamID, p.IPAddr,
		p.CommunityVisibilityState, p.ProfileState, p.PersonaName, p.ProfileURL,
		p.Avatar, p.AvatarMedium, p.AvatarFull, p.AvatarHash, p.PersonaState, p.RealName, p.TimeCreated,
		p.LocCountryCode, p.LocStateCode, p.LocCityID)
	if err != nil {
		return DBErr(err)
	}
	return nil
}

func GetPersonBySteamID(sid steamid.SID64) (model.Person, error) {
	const q = `SELECT * FROM person WHERE steam_id = $1`
	var p model.Person
	if !sid.Valid() {
		return p, ErrNoResult
	}
	err := db.Get(&p, q, sid)
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

func GetOrCreatePersonBySteamID(sid steamid.SID64) (model.Person, error) {
	const q = `SELECT * FROM person WHERE steam_id = $1`
	var p model.Person
	err := db.Get(&p, q, sid)
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

func GetBanNet(ip string) (model.BanNet, error) {
	addr := net.ParseIP(ip)
	const q = `SELECT * FROM ban_net`
	var nets []model.BanNet
	if err := db.Select(&nets, q); err != nil {
		return model.BanNet{}, DBErr(err)
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
		UPDATE ban_net SET cidr = :cidr, source = :source, updated_on = :updated_on, until = :until
		WHERE net_id = :net_id`
	if _, err := db.NamedExec(q, banNet); err != nil {
		return err
	}
	return nil
}

func insertBanNet(banNet *model.BanNet) error {
	const q = `
		INSERT INTO ban_net (cidr, source, created_on, updated_on, reason, until) 
		VALUES (:cidr, :source, :created_on, :updated_on, :reason, :until)`
	res, err := db.NamedExec(q, banNet)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	banNet.NetID = id
	return nil
}

func SaveBanNet(banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return updateBanNet(banNet)
	}
	return insertBanNet(banNet)
}

func DropNetBan(ban model.BanNet) error {
	const q = `DELETE FROM ban_net WHERE net_id = :net_id`
	if _, err := db.NamedExec(q, ban); err != nil {
		return DBErr(err)
	}
	return nil
}

func GetExpiredBans() ([]model.Ban, error) {
	const q = `SELECT * FROM ban WHERE until < $1`
	var bans []model.Ban
	if err := db.Select(&bans, q, time.Now().Unix()); err != nil {
		return nil, err
	}
	return bans, nil
}

type SearchQueryOpts struct {
	SearchTerm string
	QueryOpts
}

func GetBans(o SearchQueryOpts) ([]model.BannedPerson, error) {
	//goland:noinspection SqlResolve
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
	if err := db.Select(&bans, q2, o.OrderBy, o.Limit, o.Offset); err != nil {
		return nil, err
	}
	return bans, nil
}

func GetBansTotal() int {
	var c int
	if err := db.QueryRowx(`SELECT count(ban_id) FROM ban`).Scan(&c); err != nil {
		return 0
	}
	return c
}

func GetBansOlderThan(o QueryOpts, t time.Time) ([]model.Ban, error) {
	const q = `SELECT * FROM ban WHERE updated_on < $1 LIMIT $2 OFFSET $3`
	var bans []model.Ban
	if err := db.Select(&bans, q, t.Unix(), o.Limit, o.Offset); err != nil {
		return nil, err
	}
	return bans, nil
}

func GetExpiredNetBans() ([]model.BanNet, error) {
	const q = `SELECT * FROM ban_net WHERE until < $1`
	var bans []model.BanNet
	if err := db.Select(&bans, q, time.Now().Unix()); err != nil {
		return nil, err
	}
	return bans, nil
}

func GetFilteredWords() ([]string, error) {
	const q = `SELECT word FROM filtered_word`
	var words []string
	if err := db.Select(&words, q); err != nil {
		return nil, err
	}
	return words, nil
}

func SaveFilteredWord(word string) error {
	const q = `INSERT INTO filtered_word (word) VALUES ($1)`
	if _, err := db.Exec(q, word); err != nil {
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
	if err := db.Get(&stats, q); err != nil {
		log.Errorf("Failed to fetch stats: %v", err)
		return model.Stats{}, DBErr(err)
	}
	return stats, nil

}

func DBErr(err error) error {
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		if sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrDuplicate
		}
	}
	if err.Error() == "sql: no rows in result set" {
		return ErrNoResult
	}
	return err
}

func UpdateIndex() error {
	const q = "INSERT INTO ban_search (ban_id, steam_id, personaname, reasontext) VALUES ($1, $2, $3, $4)"
	o := NewSearchQueryOpts("")
	o.Limit = 1000000
	bans, err := GetBans(o)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM ban_search")
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	p, err := tx.Prepare(q)
	if err != nil {
		return err
	}
	for _, b := range bans {
		if _, err := p.Exec(b.BanID, b.SteamID, b.PersonaName, b.ReasonText); err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to rollback")
			}
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
