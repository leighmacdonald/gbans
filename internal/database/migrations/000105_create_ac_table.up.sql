BEGIN;

DO
$$
    BEGIN
        CREATE TYPE detection_type AS ENUM (
            'unknown',
            'silent_aim',
            'aim_snap',
            'too_many_conn',
            'interp',
            'bhop',
            'cmdnum_spike',
            'eye_angles',
            'invalid_user_cmd',
            'oob_cvar',
            'cheat_cvar'
            );
    EXCEPTION
        WHEN duplicate_object THEN null;
    END
$$;

CREATE TABLE IF NOT EXISTS anticheat
(
    anticheat_id bigint primary key GENERATED ALWAYS AS IDENTITY,
    steam_id     bigint         not null references person (steam_id) ON DELETE CASCADE,
    name         text           not null,
    detection    detection_type not null,
    summary      text           not null,
    demo_id      int references demo (demo_id) ON DELETE RESTRICT, -- Make sure we dont rm demos tied to a cheating incident
    demo_name    text           not null default '',
    server_id    int references server (server_id) ON DELETE RESTRICT,
    raw_log      text           not null,
    created_on   timestamptz    not null unique                    -- Used to uniquely identify a record
);

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS ssh_stac_path_fmt text not null default '~/srcds-%s/tf/addons/sourcemod/logs/stac';

COMMIT;
