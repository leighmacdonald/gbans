begin;

ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS weapon smallint;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS damage smallint;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS attacker_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS victim_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS assister_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS item smallint;

commit;
