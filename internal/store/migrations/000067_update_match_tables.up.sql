BEGIN;

DROP TABLE IF EXISTS match_team;

TRUNCATE TABLE match CASCADE;

ALTER TABLE match ADD column match_raw JSONB NOT NULL DEFAULT '{}';
ALTER TABLE match ADD column score_red INT NOT NULL DEFAULT 0;
ALTER TABLE match ADD column score_blu INT NOT NULL DEFAULT 0;
ALTER TABLE match ADD COLUMN time_end TIMESTAMPTZ;
ALTER TABLE match DROP COLUMN IF EXISTS winner;

ALTER TABLE match_player DROP COLUMN IF EXISTS hits;
ALTER TABLE match_player DROP COLUMN IF EXISTS shots;
ALTER TABLE match_player DROP COLUMN IF EXISTS damage;
ALTER TABLE match_player DROP COLUMN IF EXISTS kills;
ALTER TABLE match_player DROP COLUMN IF EXISTS healing;

ALTER TABLE match_player DROP COLUMN IF EXISTS backstabs;
ALTER TABLE match_player DROP COLUMN IF EXISTS headshots;
ALTER TABLE match_player DROP COLUMN IF EXISTS airshots;
ALTER TABLE match_player ADD COLUMN IF NOT EXISTS deaths INT NOT NULL DEFAULT 0;
ALTER TABLE match_player ADD COLUMN IF NOT EXISTS captures_blocked INT NOT NULL DEFAULT 0;

ALTER TABLE match_player ADD COLUMN IF NOT EXISTS player_classes INT[][2] NOT NULL DEFAULT ARRAY[]::INT[][];
ALTER TABLE match_player ADD COLUMN IF NOT EXISTS killstreaks INT[] NOT NULL DEFAULT ARRAY[]::INT[];

ALTER TABLE match_player
    ALTER COLUMN match_player_id TYPE BIGINT USING match_player_id::bigint;

ALTER TABLE match_medic DROP COLUMN charges;
ALTER TABLE match_medic DROP COLUMN avg_time_to_build;
ALTER TABLE match_medic DROP COLUMN avg_time_before_use;
ALTER TABLE match_medic DROP COLUMN death_after_charge;
ALTER TABLE match_medic ADD COLUMN charge_uber INT NOT NULL DEFAULT 0;
ALTER TABLE match_medic ADD COLUMN charge_kritz INT NOT NULL DEFAULT 0;
ALTER TABLE match_medic ADD COLUMN charge_vacc INT NOT NULL DEFAULT 0;
ALTER TABLE match_medic ADD COLUMN charge_quickfix INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS weapon (
    weapon_id SERIAL PRIMARY KEY ,
    name TEXT UNIQUE
);

create table if not exists match_weapon (
    player_weapon_id BIGSERIAL PRIMARY KEY,
    match_player_id BIGINT REFERENCES match_player (match_player_id) ON DELETE CASCADE ON UPDATE CASCADE,
    weapon_id INTEGER REFERENCES weapon (weapon_id) ON DELETE CASCADE ON UPDATE CASCADE,
    kills INTEGER NOT NULL DEFAULT 0,
    damage INTEGER NOT NULL DEFAULT 0,
    shots INTEGER NOT NULL DEFAULT 0,
    hits INTEGER NOT NULL DEFAULT 0,
    backstabs INTEGER NOT NULL DEFAULT 0,
    headshots INTEGER NOT NULL DEFAULT 0,
    airshots INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX ON match_medic (steam_id);
CREATE INDEX ON match_medic (match_id);
CREATE INDEX ON match_player (match_id);
CREATE INDEX ON match_player (steam_id);

COMMIT;
