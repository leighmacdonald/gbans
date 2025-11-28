ALTER TABLE server
    DROP COLUMN IF EXISTS discord_seed_role_ids;

ALTER TABLE config
    DROP COLUMN IF EXISTS discord_seed_channel_id;