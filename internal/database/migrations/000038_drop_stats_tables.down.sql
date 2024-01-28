begin;

create table if not exists stats_global_alltime
(
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null,

    -- Global specific
    unique_players     bigint default 0 not null

);

create table if not exists stats_global_monthly
(
    year               bigint           not null,
    month              bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null,

    -- Global specific
    unique_players     bigint default 0 not null

);
create unique index if not exists stats_global_monthly_uindex on stats_global_monthly (year, month);

create table if not exists stats_global_weekly
(
    year               bigint           not null,
    week               bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null,

    -- Global specific
    unique_players     bigint default 0 not null

);
create unique index if not exists stats_global_weekly_uindex on stats_global_weekly (year, week);

create table if not exists stats_global_daily
(
    year               bigint           not null,
    day                bigint           not null,

    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null,

    -- Global specific
    unique_players     bigint default 0 not null

);
create unique index if not exists stats_global_daily_uindex on stats_global_daily (year, day);

create table if not exists stats_player_alltime
(
    steam_id           bigint           not null
        constraint stats_player_alltime_person_steam_id_fk references person,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null

);

create unique index if not exists stats_player_alltime_uindex on stats_player_alltime (steam_id);

create table if not exists stats_player_monthly
(
    steam_id           bigint           not null
        constraint stats_player_alltime_person_steam_id_fk references person,
    year               bigint           not null,
    month              bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null

);

create unique index if not exists stats_player_monthly_uindex on stats_player_monthly (steam_id, year, month);

create table if not exists stats_player_weekly
(
    steam_id           bigint           not null
        constraint stats_player_alltime_person_steam_id_fk references person,
    year               bigint           not null,
    week               bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null

);

create unique index if not exists stats_player_weekly_uindex on stats_player_weekly (steam_id, year, week);

create table if not exists stats_player_daily
(
    steam_id           bigint           not null
        constraint stats_player_alltime_person_steam_id_fk
            references person,
    year               bigint           not null,
    day                bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null,

    -- Player specific
    deaths             bigint default 0 not null,
    games              bigint default 0 not null,
    wins               bigint default 0 not null,
    losses             bigint default 0 not null,
    damage_taken       bigint default 0 not null,
    dominated          bigint default 0 not null

);

create unique index if not exists stats_player_daily_uindex on stats_player_daily (steam_id, year, day);

create table if not exists stats_server_alltime
(
    server_id          bigint           not null
        constraint stats_server_alltime_fk
            references server,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_server_alltime_uindex on stats_server_alltime (server_id);

create table if not exists stats_server_monthly
(
    server_id          bigint           not null
        constraint stats_server_weekly_fk
            references server,
    year               bigint           not null,
    month              bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index stats_server_monthly_uindex on stats_server_monthly (server_id, year, month);

create table if not exists stats_server_weekly
(
    server_id          bigint           not null
        constraint stats_server_weekly_fk
            references server,
    year               bigint           not null,
    week               bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_server_weekly_uindex on stats_server_weekly (server_id, year, week);

create table if not exists stats_server_daily
(
    server_id          bigint           not null
        constraint stats_server_daily_fk
            references server,
    year               bigint           not null,
    day                bigint           not null,

    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_server_daily_uindex on stats_server_daily (server_id, year, day);

create table if not exists stats_map_alltime
(
    map_name           varchar          not null primary key,

    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists tats_server_map_uindex on stats_map_alltime (map_name);

create table if not exists stats_map_monthly
(
    map_name           varchar          not null primary key,
    year               bigint           not null,
    month              bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_map_monthly_uindex on stats_map_monthly (map_name, year, month);

create table if not exists stats_map_weekly
(
    map_name           varchar          not null primary key,
    year               bigint           not null,
    week               bigint           not null,
    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_map_weekly_uindex on stats_map_weekly (map_name, year, week);

create table if not exists stats_map_daily
(
    map_name           varchar          not null primary key,
    year               bigint           not null,
    day                bigint           not null,

    -- Common stats
    kills              bigint default 0 not null,
    assists            bigint default 0 not null,
    damage             bigint default 0 not null,
    healing            bigint default 0 not null,
    shots              bigint default 0 not null,
    hits               bigint default 0 not null,
    suicides           bigint default 0 not null,
    extinguishes       bigint default 0 not null,
    point_captures     bigint default 0 not null,
    point_defends      bigint default 0 not null,
    medic_dropped_uber bigint default 0 not null,
    object_built       bigint default 0 not null,
    object_destroyed   bigint default 0 not null,
    messages           bigint default 0 not null,
    messages_team      bigint default 0 not null,

    pickup_ammo_large  bigint default 0 not null,
    pickup_ammo_medium bigint default 0 not null,
    pickup_ammo_small  bigint default 0 not null,
    pickup_hp_large    bigint default 0 not null,
    pickup_hp_medium   bigint default 0 not null,
    pickup_hp_small    bigint default 0 not null,

    spawn_scout        bigint default 0 not null,
    spawn_soldier      bigint default 0 not null,
    spawn_pyro         bigint default 0 not null,
    spawn_demo         bigint default 0 not null,
    spawn_heavy        bigint default 0 not null,
    spawn_engineer     bigint default 0 not null,
    spawn_medic        bigint default 0 not null,
    spawn_sniper       bigint default 0 not null,
    spawn_spy          bigint default 0 not null,

    dominations        bigint default 0 not null,
    revenges           bigint default 0 not null,
    playtime           bigint default 0 not null,
    event_count        bigint default 0 not null

);

create unique index if not exists stats_map_daily_uindex on stats_map_daily (map_name, year, day);

commit;