begin;

ALTER TABLE IF EXISTS server_log
    DROP COLUMN IF EXISTS player_team;
ALTER TABLE IF EXISTS server_log
    DROP COLUMN IF EXISTS meta_data;
ALTER TABLE IF EXISTS server_log
    DROP COLUMN IF EXISTS healing;

commit;