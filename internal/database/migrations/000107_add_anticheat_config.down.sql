BEGIN;

ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_enabled;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_action;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_duration;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_aim_snap;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_psilent;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_bhop;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_fake_ang;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_cmd_num;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_too_many_connections;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_cheat_cvar;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_oob_var;
ALTER TABLE config
    DROP COLUMN IF EXISTS anticheat_max_invalid_user_cmd;
ALTER TABLE config
    DROP COLUMN IF EXISTS discord_anticheat_channel_id;

DROP TYPE IF EXISTS config_action;

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS discord_unregister_on_start bool not null default false;

COMMIT;
