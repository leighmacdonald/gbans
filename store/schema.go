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
	steam_id integer
		constraint player_pk
			primary key,
	created_on integer not null,
	updated_on integer not null,
	ip_addr text default '' not null,
	communityvisibilitystate int default 0 not null,
	profilestate int not null,
	personaname text not null,
	profileurl text not null,
	avatar text not null,
	avatarmedium text not null,
	avatarfull text not null,
	avatarhash text not null,
	personastate int not null,
	realname text not null,
	timecreated int not null,
	loccountrycode text not null,
	locstatecode text not null,
	loccityid int not null
);

CREATE INDEX if not exists idx_personaname_lower
ON person(LOWER(personaname));

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
	reserved_slots integer not null,
	created_on integer not null,
	updated_on integer not null,
	password text default '' not null
);

create unique index if not exists server_name_uindex
	on server (short_name);

CREATE VIRTUAL TABLE ban_search
USING fts5(ban_id, steam_id, personaname, reasontext);
`
