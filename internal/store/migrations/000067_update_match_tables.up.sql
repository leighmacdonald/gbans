BEGIN;

DROP TABLE IF EXISTS match_team;

TRUNCATE TABLE match cascade;

ALTER TABLE match ADD column match_raw jsonb not null default '{}';
ALTER TABLE match ADD column score_red int not null default 0;
ALTER TABLE match ADD column score_blu int not null default 0;
ALTER TABLE match ADD COLUMN time_end timestamptz;
ALTER TABLE match DROP COLUMN IF EXISTS winner;

ALTER TABLE match_player DROP COLUMN IF EXISTS hits;
ALTER TABLE match_player DROP COLUMN IF EXISTS shots;
ALTER TABLE match_player DROP COLUMN IF EXISTS damage;

ALTER TABLE match_player DROP COLUMN IF EXISTS backstabs;
ALTER TABLE match_player DROP COLUMN IF EXISTS headshots;
ALTER TABLE match_player DROP COLUMN IF EXISTS airshots;

ALTER TABLE match_medic DROP COLUMN charges;
ALTER TABLE match_medic DROP COLUMN avg_time_to_build;
ALTER TABLE match_medic DROP COLUMN avg_time_before_use;
ALTER TABLE match_medic DROP COLUMN death_after_charge;
ALTER TABLE match_medic ADD column charge_uber int not null default 0;
ALTER TABLE match_medic ADD column charge_kritz int not null default 0;
ALTER TABLE match_medic ADD column charge_vacc int not null default 0;
ALTER TABLE match_medic ADD column charge_quickfix int not null default 0;


create table if not exists weapon (
    weapon_id serial primary key,
    name text unique
);

create table if not exists match_weapon (
    match_weapon_id bigserial primary key,
    match_id INTEGER REFERENCES match (match_id) ON DELETE CASCADE ON UPDATE CASCADE,
    weapon_id INTEGER REFERENCES weapon (weapon_id) ON DELETE CASCADE ON UPDATE CASCADE,
    kills INTEGER NOT NULL default 0,
    damage INTEGER NOT NULL default 0,
    shots INTEGER NOT NULL default 0,
    hits INTEGER not null default 0,
    backstabs INTEGER not null default 0,
    headshots INTEGER not null default 0,
    airshots INTEGER not null default 0
);

COMMIT;
