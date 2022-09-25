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

alter table if exists ban alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists ban alter column updated_on type timestamptz using created_on::timestamptz;
alter table if exists ban alter column valid_until type timestamptz using valid_until::timestamptz;

alter table if exists ban_asn alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists ban_asn alter column updated_on type timestamptz using created_on::timestamptz;
alter table if exists ban_asn alter column valid_until type timestamptz using valid_until::timestamptz;

alter table if exists ban_net alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists ban_net alter column updated_on type timestamptz using created_on::timestamptz;
alter table if exists ban_net alter column valid_until type timestamptz using valid_until::timestamptz;

alter table if exists ban_group alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists ban_group alter column updated_on type timestamptz using created_on::timestamptz;
alter table if exists ban_group alter column valid_until type timestamptz using valid_until::timestamptz;

alter table if exists ban_appeal alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists ban_appeal alter column updated_on type timestamptz using updated_on::timestamptz;

alter table if exists filtered_word alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists filtered_word alter column discord_created_on type timestamptz using discord_created_on::timestamptz;

alter table if exists match alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists match_player alter column time_start type timestamptz using time_start::timestamptz;
alter table if exists match_player alter column time_end type timestamptz using time_end::timestamptz;

alter table if exists media alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists media alter column updated_on type timestamptz using updated_on::timestamptz;

alter table if exists news alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists news alter column updated_on type timestamptz using updated_on::timestamptz;

alter table if exists person alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists person alter column updated_on type timestamptz using updated_on::timestamptz;
alter table if exists person alter column updated_on type timestamptz using updated_on_steam::timestamptz;

alter table if exists person_connections alter column created_on type timestamptz using created_on::timestamptz;

alter table if exists person_messages alter column created_on type timestamptz using created_on::timestamptz;

alter table if exists report alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists report alter column updated_on type timestamptz using updated_on::timestamptz;

alter table if exists report_message alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists report_message alter column updated_on type timestamptz using updated_on::timestamptz;

alter table if exists server alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists server alter column updated_on type timestamptz using updated_on::timestamptz;
alter table if exists server alter column token_created_on type timestamptz using token_created_on::timestamptz;

alter table if exists wiki alter column created_on type timestamptz using created_on::timestamptz;
alter table if exists wiki alter column updated_on type timestamptz using updated_on::timestamptz;

COMMIT;
