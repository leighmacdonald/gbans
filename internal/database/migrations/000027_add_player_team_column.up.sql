begin;

ALTER TABLE IF EXISTS server_log
    ADD COLUMN IF NOT EXISTS player_team smallint not null default 0;
ALTER TABLE IF EXISTS server_log
    ADD COLUMN IF NOT EXISTS meta_data jsonb;
ALTER TABLE IF EXISTS server_log
    ADD COLUMN IF NOT EXISTS healing bigint not null default 0;

commit;