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
	ban_source int default 0 not null
);

create unique index if not exists ban_steam_id_uindex
	on ban (steam_id);

create table if not exists ban_net
(
	net_id integer
		constraint ban_net_pk
			primary key autoincrement,
	cidr text not null,
	source int default 0 not null,
	created_on integer not null,
	updated_on integer not null,
	reason text default '' not null,
	until integer not null
);

create unique index if not exists ban_net_cidr_uindex
	on ban_net (cidr);

create table if not exists person
(
	player_id integer
		constraint player_pk
			primary key autoincrement,
	created_on integer not null,
	updated_on integer not null,
	steam_id interger not null,
	name text default '' not null,
	ip_addr text default '' not null
);

create table if not exists player
(
	player_id integer
		constraint player_pk
			primary key autoincrement,
	created_on integer not null,
	updated_on integer not null,
	steam_id integer not null,
	name text default '' not null,
	ip_addr text default '' not null
);

create table if not exists server
(
	server_id integer
		constraint server_pk
			primary key autoincrement,
	short_name text not null,
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
	on server (short_name);


`
