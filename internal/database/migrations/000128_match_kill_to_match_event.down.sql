-- Restore the kill-only match_kill table removed by the match_event migration.
-- Event rows themselves are not recoverable on downgrade.
BEGIN;

DROP TABLE IF EXISTS match_event;

create table if not exists match_kill (
  match_kill_id serial primary key,
  match_id uuid not null references match (match_id) on delete cascade,
  tick integer not null,
  killer_steam_id bigint references person (steam_id),
  victim_steam_id bigint not null references person (steam_id),
  weapon text not null,
  killer_pos_x double precision,
  killer_pos_y double precision,
  killer_pos_z double precision,
  victim_pos_x double precision,
  victim_pos_y double precision,
  victim_pos_z double precision,
  killer_angles_pitch double precision,
  killer_angles_yaw double precision,
  victim_angles_pitch double precision,
  victim_angles_yaw double precision
);

create index if not exists idx_match_kill_match_id on match_kill (match_id, tick);

ALTER TABLE IF EXISTS match_kill
  ADD COLUMN IF NOT EXISTS round_id integer
    CONSTRAINT match_kill_round_round_id_fk
    REFERENCES match_round (round_id)
    ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE IF EXISTS match_kill
  ADD COLUMN IF NOT EXISTS killer_position geometry(PointZ)
    GENERATED ALWAYS AS (
      CASE
        WHEN killer_pos_x IS NULL OR killer_pos_y IS NULL OR killer_pos_z IS NULL THEN NULL
        ELSE ST_MakePoint(killer_pos_x, killer_pos_y, killer_pos_z)
      END
    ) STORED;

ALTER TABLE IF EXISTS match_kill
  ADD COLUMN IF NOT EXISTS victim_position geometry(PointZ)
    GENERATED ALWAYS AS (
      CASE
        WHEN victim_pos_x IS NULL OR victim_pos_y IS NULL OR victim_pos_z IS NULL THEN NULL
        ELSE ST_MakePoint(victim_pos_x, victim_pos_y, victim_pos_z)
      END
    ) STORED;

COMMENT ON COLUMN match_kill.round_id IS 'Owning round when a kill tick falls within a parsed round interval; NULL when unknown.';
COMMENT ON COLUMN match_kill.killer_position IS 'Game-local Cartesian coordinates in X, Y, Z order; NULL for world kills or missing positions.';
COMMENT ON COLUMN match_kill.victim_position IS 'Game-local Cartesian coordinates in X, Y, Z order; NULL when the victim position is missing.';

CREATE INDEX IF NOT EXISTS idx_match_kill_round_id
  ON match_kill (round_id);

CREATE INDEX IF NOT EXISTS idx_match_kill_killer_position
  ON match_kill USING GIST (killer_position);

CREATE INDEX IF NOT EXISTS idx_match_kill_victim_position
  ON match_kill USING GIST (victim_position);

COMMIT;
