BEGIN;

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


ALTER TABLE match DROP column if exists match_raw;
ALTER TABLE match DROP column if exists winner;

ALTER TABLE match_medic ADD column charges int not null default 0;
ALTER TABLE match_medic DROP COLUMN charge_uber;
ALTER TABLE match_medic DROP COLUMN charge_kritz;
ALTER TABLE match_medic DROP COLUMN charge_vacc;
ALTER TABLE match_medic DROP COLUMN charge_quickfix;

DROP TABLE IF EXISTS match_weapon;

DROP TABLE IF EXISTS weapon;

COMMIT;
