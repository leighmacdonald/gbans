BEGIN;

ALTER TABLE config DROP COLUMN discord_kick_log_channel_id;
ALTER TABLE config ADD COLUMN general_steam_key text not null default '';

COMMIT;
