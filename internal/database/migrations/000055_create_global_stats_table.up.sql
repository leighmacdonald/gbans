BEGIN;

CREATE TABLE global_stats_players
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
    created_on        timestamp            not null
);

COMMIT;
