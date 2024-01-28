begin;

ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS payload;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS weapon smallint;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS damage smallint;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS attacker_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS victim_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS assister_position geometry(POINTZ);
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS item smallint;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS extra text default '' not null;
ALTER TABLE IF EXISTS server_log ADD COLUMN IF NOT EXISTS player_class int default 0;

commit;
