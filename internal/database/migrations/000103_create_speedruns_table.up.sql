BEGIN;

CREATE TABLE IF NOT EXISTS map
(
    map_id     int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    map_name   text        not null unique,
    updated_on timestamptz not null default now(),
    created_on timestamptz not null default now()
);

CREATE INDEX IF NOT EXISTS map_name_idx ON map (map_name);

CREATE TABLE IF NOT EXISTS speedrun
(
    speedrun_id  int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    map_id       int references map (map_id) DEFERRABLE INITIALLY DEFERRED       NOT NULL,
    server_id    int references server (server_id) DEFERRABLE INITIALLY DEFERRED NOT NULL,
    category     text                                                            NOT NULL,
    duration     interval                                                        not null,
    player_count int                                                             not null,
    bot_count    int                                                             not null,
    initial_rank int                                                             not null,
    created_on   timestamptz                                                     not null default now()
);

CREATE TABLE IF NOT EXISTS speedrun_runners
(
    speedrun_id int REFERENCES speedrun (speedrun_id) ON DELETE CASCADE,
    steam_id    bigint REFERENCES person (steam_id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED NOT NULL,
    duration    interval                                                                             not null
);

CREATE TABLE IF NOT EXISTS speedrun_capture
(
    speedrun_id int REFERENCES speedrun (speedrun_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    round_id    int      not null,
    duration    interval not null,
    point_name  text     not null default '',
    primary key (speedrun_id, round_id)
);

CREATE TABLE IF NOT EXISTS speedrun_capture_runners
(
    speedrun_id int REFERENCES speedrun (speedrun_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    round_id    int      not null,
    steam_id    bigint   not null references person (steam_id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    duration    interval not null,
    primary key (speedrun_id, round_id, steam_id)
);

ALTER TABLE config
    ADD COLUMN general_speedruns_enabled bool not null default false;

COMMIT;
