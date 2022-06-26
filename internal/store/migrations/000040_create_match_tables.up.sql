begin;

create table match
(
    match_id   serial
        constraint match_pk
            primary key,
    server_id  integer   not null
        constraint match_server_server_id_fk
            references server,
    map        text      not null,
    created_on timestamp not null,
    title      text      not null
);

create table match_medic
(
    match_medic_id         serial
        constraint match_medic_pk
            primary key,
    match_id               integer                    not null
        constraint match_medic_match_match_id_fk
            references match
            on update cascade on delete cascade,
    steam_id               bigint                     not null
        constraint match_medic_steam_id_fk
            references person,
    healing                integer          default 0 not null,
    charges                integer          default 0 not null,
    drops                  integer          default 0 not null,
    avg_time_to_build      integer          default 0 not null,
    avg_time_before_use    integer          default 0 not null,
    near_full_charge_death integer          default 0 not null,
    avg_uber_length        double precision default 0 not null,
    death_after_charge     integer          default 0 not null,
    major_adv_lost         integer          default 0 not null,
    biggest_adv_lost       integer          default 0 not null
);

create unique index match_medic_match_id_steam_id_uindex
    on match_medic (match_id, steam_id);

create table match_player
(
    match_player_id     serial
        constraint match_player_pk
            primary key,
    match_id            integer           not null
        constraint match_player_match_match_id_fk
            references match
            on update cascade on delete cascade,
    steam_id            bigint            not null
        constraint match_player_person_steam_id_fk
            references person
            on update restrict on delete restrict,
    team                integer default 0 not null,
    time_start          timestamp         not null,
    time_end            timestamp         not null,
    kills               integer default 0 not null,
    assists             integer default 0 not null,
    deaths              integer default 0 not null,
    dominations         integer default 0 not null,
    dominated           integer default 0 not null,
    revenges            integer default 0 not null,
    damage              integer default 0 not null,
    damage_taken        integer default 0 not null,
    healing             integer default 0 not null,
    healing_taken       integer default 0 not null,
    health_packs        integer default 0 not null,
    backstabs           integer default 0 not null,
    headshots           integer default 0 not null,
    airshots            integer default 0 not null,
    captures            integer default 0 not null,
    shots               integer default 0 not null,
    extinguishes        integer default 0 not null,
    hits                integer default 0 not null,
    buildings           integer default 0 not null,
    buildings_destroyed integer default 0 not null
);

create unique index match_player_steam_id_uindex
    on match_player (match_id, steam_id);

create table match_team
(
    match_team_id serial
        constraint match_team_pk
            primary key,
    match_id      integer           not null
        constraint match_team_match_match_id_fk
            references match
            on update cascade on delete cascade,
    team          integer default 0 not null,
    kills         integer default 0 not null,
    damage        integer default 0 not null,
    charges       integer default 0 not null,
    drops         integer default 0 not null,
    caps          integer default 0 not null,
    mid_fights    integer default 0 not null
);

create unique index match_team_uindex
    on match_team (match_id, team);

commit;
