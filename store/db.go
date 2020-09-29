package store

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"log"
	"net"
	"time"
)

var (
	db           *sqlx.DB
	ErrNoResult  = errors.New("No results found")
	ErrDuplicate = errors.New("Duplicate entity")
)

func Init(path string) {
	db = sqlx.MustConnect("sqlite3", path)
	db.MustExec(schema)
	_, err := GetOrCreatePlayerBySteamID(0)
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
			token_created_on, created_on, updated_on 
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
			token_created_on, created_on, updated_on 
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
			token_created_on, created_on, updated_on 
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
		    short_name, token, address, port, rcon, token_created_on, created_on, updated_on, password) 
		VALUES (:short_name, :token, :address, :port, :rcon, :token_created_on, :created_on, :updated_on, :password);`
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
		    rcon = :rcon, token_created_on = :token_created_on, updated_on = :updated_on
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

func SavePlayer(player *model.Person) error {
	player.UpdatedOn = time.Now().Unix()
	if player.PlayerID > 0 {
		return updatePlayer(player)
	}
	player.CreatedOn = player.UpdatedOn
	return insertPlayer(player)
}

func updatePlayer(player *model.Person) error {
	const q = `
		UPDATE person
		SET name = :name, steam_id = :steam_id, updated_on = :updated_on
		WHERE player_id = :player_id`
	if _, err := db.NamedExec(q, player); err != nil {
		return DBErr(err)
	}
	return nil
}

func insertPlayer(player *model.Person) error {
	const q = `
		INSERT INTO person (name, created_on, updated_on, steam_id) 
		VALUES (:name, :created_on, :updated_on, :steam_id)`
	res, err := db.NamedExec(q, player)
	if err != nil {
		return DBErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return DBErr(err)
	}
	player.PlayerID = id
	return nil
}

func GetOrCreatePlayerBySteamID(sid steamid.SID64) (model.Person, error) {
	const q = `SELECT * FROM player WHERE steam_id = $1`
	var p model.Person
	err := db.Get(&p, q, sid)
	if err != nil && DBErr(err) == ErrNoResult {
		p.SteamID = sid
		if err := SavePlayer(&p); err != nil {
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

func GetExpiredNetBans() ([]model.BanNet, error) {
	const q = `SELECT * FROM ban_net WHERE until < $1`
	var bans []model.BanNet
	if err := db.Select(&bans, q, time.Now().Unix()); err != nil {
		return nil, err
	}
	return bans, nil
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
