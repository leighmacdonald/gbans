BEGIN;

CREATE TABLE contest
(
    contest_id           uuid primary key     default gen_random_uuid(),
    title                text        not null unique,
    public               bool        not null default true,
    hide_submissions     bool        not null default false,
    description          text        not null,
    date_start           timestamptz not null,
    date_end             timestamptz not null,
    max_submissions      int         not null default 1,
    media_types          text        not null default '',
    deleted              bool        not null default false,
    voting               bool        not null default false,
    min_permission_level int         not null default 10,
    down_votes           bool        not null default false,
    created_on           timestamptz not null,
    updated_on           timestamptz not null
);

CREATE TABLE contest_entry
(
    contest_entry_id uuid primary key     default gen_random_uuid(),
    contest_id       uuid        not null references contest on delete cascade on update cascade,
    steam_id         bigint      not null references person,
    asset_id         uuid        not null references asset,
    description      text        not null default '',
    placement        int         not null default 0,
    deleted          bool        not null default false,
    created_on       timestamptz not null,
    updated_on       timestamptz not null
);

CREATE TABLE contest_entry_vote
(
    contest_entry_vote_id bigserial primary key,
    contest_entry_id      uuid        not null references contest_entry on delete cascade on update cascade,
    steam_id              bigint      not null references person,
    vote                  bool        not null,
    created_on            timestamptz not null,
    updated_on            timestamptz not null,
    UNIQUE (contest_entry_id, steam_id)
);

DROP TABLE IF EXISTS global_stats_players;
DROP TABLE IF EXISTS global_stats_players_daily;
DROP TABLE IF EXISTS global_stats_players_hourly;
DROP TABLE IF EXISTS local_stats_players;
DROP TABLE IF EXISTS local_stats_players_daily;
DROP TABLE IF EXISTS local_stats_players_hourly;
DROP TABLE IF EXISTS stats_global_alltime;
DROP TABLE IF EXISTS stats_map_alltime;
DROP TABLE IF EXISTS stats_player_alltime;
DROP TABLE IF EXISTS stats_server_alltime;

COMMIT;
