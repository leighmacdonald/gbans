BEGIN;

CREATE TABLE local_stats_players
(
    stat_id          bigserial primary key,
    players          integer default 0    not null,
    capacity_full    integer default 0    not null,
    capacity_empty   integer default 0    not null,
    capacity_partial integer default 0    not null,
    regions          jsonb   default '{}' not null,
    map_types         jsonb   default '{}' not null,
    created_on       timestamptz          not null
);

CREATE TABLE local_stats_players_hourly
(
    stat_id          bigserial primary key,
    players          integer default 0    not null,
    capacity_full    integer default 0    not null,
    capacity_empty   integer default 0    not null,
    capacity_partial integer default 0    not null,
    regions          jsonb   default '{}' not null,
    map_types         jsonb   default '{}' not null,
    created_on       timestamptz          not null
);

alter table local_stats_players_hourly
    add constraint local_stats_players_hourly_created_on_uidx
        unique (created_on);

CREATE TABLE local_stats_players_daily
(
    stat_id          bigserial primary key,
    players          integer default 0    not null,
    capacity_full    integer default 0    not null,
    capacity_empty   integer default 0    not null,
    capacity_partial integer default 0    not null,
    regions          jsonb   default '{}' not null,
    map_types         jsonb   default '{}' not null,
    created_on       timestamptz          not null
);

alter table local_stats_players_daily
    add constraint local_stats_players_daily_created_on_uidx
        unique (created_on);

COMMIT;
