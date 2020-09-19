package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
)

var (
	db *sqlx.DB
)

const schema = `
create table if not exists reason
(
	reason_id integer
		constraint reason_pk
			primary key autoincrement,
	reason text not null
);

create table if not exists ban
(
	ban_id integer
		constraint ban_pk
			primary key autoincrement,
	steam_id INTEGER not null,
	reason_id integer
		references reason
			on update cascade on delete restrict,
	note text default '' not null,
	created_on integer default 0 not null,
	updated_on integer default 0 not null
);

create unique index if not exists ban_steam_id_uindex
	on ban (steam_id);

create unique index if not exists reason_reason_uindex
	on reason (reason);

create table if not exists server
(
	server_id integer
		constraint server_pk
			primary key autoincrement,
	name text not null,
	token text,
	address text not null,
	port integer not null,
	rcon text not null,
	token_created_on integer,
	created_on integer not null,
	updated_on integer not null
);

create unique index if not exists server_name_uindex
	on server (name);

create unique index if not exists server_token_uindex
	on server (token);

`

type Server struct {
	ServerID       int64  `json:"server_id"`
	Name           string `json:"name"`
	Token          string `json:"token"`
	Address        string `json:"address"`
	Port           int    `json:"port"`
	RCON           string `json:"rcon"`
	TokenCreatedOn int64  `json:"token_created_on"`
	CreatedOn      int64  `json:"created_on"`
	UpdatedOn      int64  `json:"updated_on"`
}

func setupDB() error {
	if _, err := db.Exec(schema); err != nil {
		log.Errorf("Failed to setup db")
		return err
	}
	return nil
}

func getServer(serverID int) (Server, error) {
	var s Server
	const q = `
		SELECT 
		    server_id, name, token, address, port, rcon,
			token_created_on, created_on, updated_on 
		FROM server
		WHERE server_id = $1`
	if err := db.Get(&s, q, serverID); err != nil {
		return Server{}, err
	}
	return s, nil
}

func saveServer(server Server) error {
	if server.ServerID > 0 {
		return updateServer(server)
	}
	return saveServer(server)
}

func insertServer(server Server) error {
	return nil
}

func updateServer(server Server) error {
	return nil
}
