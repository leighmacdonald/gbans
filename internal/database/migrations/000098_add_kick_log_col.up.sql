BEGIN;

ALTER TABLE config ADD COLUMN discord_kick_log_channel_id text not null default '';
ALTER TABLE config DROP COLUMN general_steam_key;

COMMIT;
