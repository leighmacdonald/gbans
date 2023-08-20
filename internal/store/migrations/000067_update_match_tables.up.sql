BEGIN;

DROP TABLE IF EXISTS match_player_killstreak CASCADE;
DROP TABLE IF EXISTS match_weapon CASCADE;
DROP TABLE IF EXISTS match_team CASCADE;
DROP TABLE IF EXISTS match_player_class CASCADE;
DROP TABLE IF EXISTS match_medic CASCADE;
DROP TABLE IF EXISTS match_player CASCADE;
DROP TABLE IF EXISTS match CASCADE;

create table match
(
    match_id   uuid
        constraint match_pk
            primary key,
    server_id  integer                  not null
        constraint match_server_server_id_fk
            references server,
    map        text                     not null
        constraint map_not_empty_check
            check (map <> ''::text),
    title      text                     not null,
    score_red  integer default 0        not null,
    score_blu  integer default 0        not null,
    time_red   integer default 0        not null,
    time_blu   integer default 0        not null,
    winner     integer default 0        not null,
    time_start timestamp with time zone not null,
    time_end   timestamp with time zone not null
);

create table match_player
(
    match_player_id bigserial                not null
        constraint match_player_pk
            primary key,
    match_id        uuid                     not null
        constraint match_player_match_match_id_fk
            references match
            on update cascade on delete cascade,
    steam_id        bigint                   not null
        constraint match_player_person_steam_id_fk
            references person
            on update restrict on delete restrict,
    team            integer default 0        not null,
    time_start      timestamp with time zone not null,
    time_end        timestamp with time zone not null,
    health_packs    integer default 0        not null,
    extinguishes    integer default 0        not null,
    buildings       integer default 0        not null
);

create unique index match_player_steam_id_uindex
    on match_player (match_id, steam_id);

create index match_player_steam_id_idx
    on match_player (steam_id);

create index match_player_match_id_idx
    on match_player (match_id);

create table match_medic
(
    match_medic_id         serial
        constraint match_medic_pk
            primary key,
    match_player_id        BIGINT REFERENCES match_player (match_player_id) ON DELETE CASCADE ON UPDATE CASCADE,
    healing                integer          default 0 not null,
    drops                  integer          default 0 not null,
    near_full_charge_death integer          default 0 not null,
    avg_uber_length        double precision default 0 not null,
    major_adv_lost         integer          default 0 not null,
    biggest_adv_lost       integer          default 0 not null,
    charge_uber            integer          default 0 not null,
    charge_kritz           integer          default 0 not null,
    charge_vacc            integer          default 0 not null,
    charge_quickfix        integer          default 0 not null
);

CREATE TABLE IF NOT EXISTS weapon
(
    weapon_id SERIAL PRIMARY KEY,
    name      TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS match_weapon
(
    player_weapon_id BIGSERIAL PRIMARY KEY,
    match_player_id  BIGINT REFERENCES match_player (match_player_id) ON DELETE CASCADE ON UPDATE CASCADE,
    weapon_id        INTEGER REFERENCES weapon (weapon_id) ON DELETE CASCADE ON UPDATE CASCADE,
    kills            INTEGER NOT NULL DEFAULT 0,
    damage           INTEGER NOT NULL DEFAULT 0,
    shots            INTEGER NOT NULL DEFAULT 0,
    hits             INTEGER NOT NULL DEFAULT 0,
    backstabs        INTEGER NOT NULL DEFAULT 0,
    headshots        INTEGER NOT NULL DEFAULT 0,
    airshots         INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS player_class
(
    player_class_id INT PRIMARY KEY,
    class_name      TEXT NOT NULL UNIQUE,
    class_key       TEXT NOT NULL UNIQUE
);

INSERT INTO player_class (player_class_id, class_name, class_key)
VALUES (0, 'Spectator', 'spectator'),
       (1, 'Scout', 'scout'),
       (2, 'Soldier', 'soldier'),
       (3, 'Pyro', 'pyro'),
       (4, 'Demo', 'demo'),
       (5, 'Heavy', 'heavy'),
       (6, 'Engineer', 'engineer'),
       (7, 'Medic', 'medic'),
       (8, 'Sniper', 'sniper'),
       (9, 'Spy', 'spy')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS match_player_killstreak
(
    match_killstreak_id BIGSERIAL PRIMARY KEY,
    match_player_id     BIGINT REFERENCES match_player (match_player_id) ON DELETE CASCADE ON UPDATE CASCADE,
    player_class_id     INTEGER REFERENCES player_class (player_class_id) ON DELETE CASCADE ON UPDATE CASCADE,
    killstreak          INTEGER NOT NULL DEFAULT 0,
    duration            INTEGER NOT NULL DEFAULT 0

);

CREATE TABLE IF NOT EXISTS match_player_class
(
    match_player_class_id BIGSERIAL PRIMARY KEY,
    match_player_id       BIGINT REFERENCES match_player (match_player_id) ON DELETE CASCADE ON UPDATE CASCADE,
    player_class_id       INTEGER REFERENCES player_class (player_class_id) ON DELETE CASCADE ON UPDATE CASCADE,
    kills                 INTEGER NOT NULL DEFAULT 0,
    assists               INTEGER NOT NULL DEFAULT 0,
    deaths                INTEGER NOT NULL DEFAULT 0,
    playtime              INTEGER NOT NULL DEFAULT 0,
    dominations           INTEGER NOT NULL DEFAULT 0,
    dominated             INTEGER NOT NULL DEFAULT 0,
    revenges              INTEGER NOT NULL DEFAULT 0,
    damage                INTEGER NOT NULL DEFAULT 0,
    damage_taken          INTEGER NOT NULL DEFAULT 0,
    healing_taken         INTEGER NOT NULL DEFAULT 0,
    captures              INTEGER NOT NULL DEFAULT 0,
    captures_blocked      INTEGER NOT NULL DEFAULT 0,
    buildings_destroyed   INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE IF EXISTS person_messages
    ADD IF NOT EXISTS match_id uuid;

ALTER TABLE IF EXISTS person_messages
    ADD CONSTRAINT person_messages_match_match_id_fk
        FOREIGN KEY (match_id) REFERENCES match;

COMMIT;
