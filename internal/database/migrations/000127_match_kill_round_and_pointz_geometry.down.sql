BEGIN;

DROP INDEX IF EXISTS idx_match_kill_victim_position;
DROP INDEX IF EXISTS idx_match_kill_killer_position;
DROP INDEX IF EXISTS idx_match_kill_round_id;

ALTER TABLE IF EXISTS match_kill
  DROP COLUMN IF EXISTS victim_position,
  DROP COLUMN IF EXISTS killer_position,
  DROP COLUMN IF EXISTS round_id;

COMMIT;
