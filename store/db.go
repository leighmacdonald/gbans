package store

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"time"
)

var (
	db *sqlx.DB
)

func Init(path string) {
	db = sqlx.MustConnect("sqlite3", path)
	db.MustExec(schema)
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
		    server_id, server_name, token, address, port, rcon,
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
		    server_id, server_name, token, address, port, rcon,
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
		    server_id, server_name, token, address, port, rcon,
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
		    server_id, server_name, token, address, port, rcon,
			token_created_on, created_on, updated_on 
		FROM server
		WHERE server_name = $1`
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
		    server_name, token, address, port, rcon, token_created_on, created_on, updated_on) 
		VALUES (:server_name, :token, :address, :port, :rcon, :token_created_on, :created_on, :updated_on);`
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
		SET server_name = :server_name, token = :token, address = :address, port = :port,
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

func GetBan(steamID steamid.SID64) (model.Ban, error) {
	const q = `
		SELECT b.ban_id, b.steam_id, b.ban_type, b.reason, b.note, b.created_on, b.updated_on, b.reason_text 
		FROM ban b 
		WHERE steam_id = $1`
	var b model.Ban
	if err := db.Get(&b, q, steamID.Int64()); err != nil {
		return model.Ban{}, err
	}
	return b, nil
}

func SaveBan(ban *model.Ban) error {
	if ban.BanID > 0 {
		return updateBan(ban)
	}
	return insertBan(ban)
}

func insertBan(ban *model.Ban) error {
	const q = `
		INSERT INTO ban (steam_id, author_id,ban_type, reason, reason_text, note, until, created_on, updated_on) 
		VALUES (:steam_id, :author_id, :ban_type, :reason, :reason_text, :note, :until, :created_on, :updated_on)`
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
		UPDATE ban SET ban_type = :ban_type, reason = :reason, reason_text = :reason_text, note = :note, updated_on = :updated_on 
		WHERE ban_id = :ban_id`
	if _, err := db.NamedExec(q, ban); err != nil {
		return DBErr(err)
	}
	return nil
}

func DBErr(err error) error {
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		if sqliteErr.Code == sqlite3.ErrConstraint {
			return model.ErrDuplicate
		}
	}
	if err.Error() == "sql: no rows in result set" {

	}
	return err
}
