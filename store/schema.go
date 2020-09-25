package store

const schema = `
create table if not exists ban
(
	ban_id integer
		constraint ban_pk
			primary key autoincrement,
	steam_id INTEGER not null,
	author_id INTEGER default 0 not null,
	ban_type integer not null,
	reason integer not null,
	reason_text TEXT default '' not null,
	note text default '' not null,
	until integer default 0 not null,
	created_on integer default 0 not null,
	updated_on integer default 0 not null,
	active integer default 1
);

create index if not exists ban_active_index
	on ban (active);

create unique index if not exists ban_steam_id_uindex
	on ban (steam_id);

create table if not exists server
(
	server_id integer
		constraint server_pk
			primary key autoincrement,
	server_name text not null,
	token text default '' not null,
	address text not null,
	port integer not null,
	rcon text not null,
	token_created_on integer,
	created_on integer not null,
	updated_on integer not null,
	password text default '' not null
);

create unique index if not exists server_name_uindex
	on server (server_name);

`
