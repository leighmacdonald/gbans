BEGIN;

alter table if exists global_stats_players
    alter column created_on type timestamptz using created_on::timestamptz;

ALTER TABLE IF EXISTS global_stats_players ADD COLUMN IF NOT EXISTS regions  jsonb default '{}' not null;

CREATE TABLE global_stats_players_hourly
(
    stat_id           bigserial primary key,
    players           integer default 0    not null,
    bots              integer default 0    not null,
    secure            integer default 0    not null,
    servers_community integer default 0    not null,
    servers_total     integer default 0    not null,
    capacity_full     integer default 0    not null,
    capacity_empty    integer default 0    not null,
    capacity_partial  integer default 0    not null,
    map_types         jsonb   default '{}' not null,
    regions           jsonb   default '{}' not null,
    created_on        timestamptz            not null
);

alter table global_stats_players_hourly
    add constraint global_stats_players_hourly_created_on_uidx
        unique (created_on);

CREATE TABLE global_stats_players_daily
(
    stat_id           bigserial primary key,
    players           integer default 0    not null,
    bots              integer default 0    not null,
    secure            integer default 0    not null,
    servers_community integer default 0    not null,
    servers_total     integer default 0    not null,
    capacity_full     integer default 0    not null,
    capacity_empty    integer default 0    not null,
    capacity_partial  integer default 0    not null,
    map_types         jsonb   default '{}' not null,
    regions           jsonb   default '{}' not null,
    created_on        timestamptz            not null
);

alter table global_stats_players_daily
    add constraint global_stats_players_daily_created_on_uidx
        unique (created_on);

COMMIT;
