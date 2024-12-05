BEGIN;

CREATE TABLE IF NOT EXISTS speedrun (
    speedrun_id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    map_name text NOT NULL,
    category text NOT NULL,
    duration interval,
    player_count int,
    bot_count int,
    created_on timestamptz not null default now()
);

CREATE TABLE IF NOT EXISTS speedrun_runners (
    speedrun_id int REFERENCES speedrun (speedrun_id) ON DELETE CASCADE,
    steam_id bigint REFERENCES person (steam_id),
    duration interval
);

COMMIT;
