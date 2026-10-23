begin;

-- Per-map aggregate statistics for the /maps page.
--
-- Daily is the source of truth for arbitrary rolling ranges (e.g. the last
-- 90 days). The player-grain view is required for exact distinct player
-- counts across multiple days, since daily distinct counts cannot be summed.
-- The all-time view keeps the all-time table preset fast.
--
-- Match-level metrics are aggregated before joining player rows so playtime
-- and match counts are never multiplied by the number of players in a match.

create materialized view stats_map_daily_view as
with match_players as (
  select
    r.match_id,
    p.steam_id,
    sum(p.kills)::bigint as kills,
    sum(p.assists)::bigint as assists,
    sum(p.deaths)::bigint as deaths,
    sum(p.damage)::bigint as damage,
    sum(p.healing)::bigint as healing,
    sum(p.captures)::bigint as captures,
    sum(p.captures_blocked)::bigint as captures_blocked
  from
    match_round r
    join match_round_player p using (round_id)
  group by
    r.match_id,
    p.steam_id
),
match_rounds as (
  select
    match_id,
    count(*)::bigint as rounds_played,
    count(*) filter (where is_stalemate)::bigint as stalemate_rounds
  from
    match_round
  group by
    match_id
),
match_player_totals as (
  select
    match_id,
    sum(kills)::bigint as kills,
    sum(assists)::bigint as assists,
    sum(deaths)::bigint as deaths,
    sum(damage)::bigint as damage,
    sum(healing)::bigint as healing,
    sum(captures)::bigint as captures,
    sum(captures_blocked)::bigint as captures_blocked,
    count(*)::bigint as player_appearances
  from
    match_players
  group by
    match_id
),
map_metrics as (
  select
    date_trunc('day', m.start_time) as date_bucket,
    m.stats_bucket_id,
    m.map_id,
    count(*)::bigint as matches_played,
    count(*) filter (where m.score_red <> m.score_blu)::bigint as matches_decided,
    count(*) filter (where m.score_red = m.score_blu)::bigint as matches_drawn,
    count(*) filter (where m.score_red > m.score_blu)::bigint as red_wins,
    count(*) filter (where m.score_blu > m.score_red)::bigint as blu_wins,
    sum(m.duration_ms)::bigint as total_duration_ms,
    coalesce(sum(mr.rounds_played), 0)::bigint as rounds_played,
    coalesce(sum(mr.stalemate_rounds), 0)::bigint as stalemate_rounds,
    coalesce(sum(mpt.player_appearances), 0)::bigint as player_appearances,
    coalesce(sum(mpt.kills), 0)::bigint as kills,
    coalesce(sum(mpt.assists), 0)::bigint as assists,
    coalesce(sum(mpt.deaths), 0)::bigint as deaths,
    coalesce(sum(mpt.damage), 0)::bigint as damage,
    coalesce(sum(mpt.healing), 0)::bigint as healing,
    coalesce(sum(mpt.captures), 0)::bigint as captures,
    coalesce(sum(mpt.captures_blocked), 0)::bigint as captures_blocked,
    min(m.start_time) as first_played_at,
    max(m.start_time) as last_played_at
  from
    match m
    left join match_rounds mr using (match_id)
    left join match_player_totals mpt using (match_id)
  group by
    date_bucket,
    m.stats_bucket_id,
    m.map_id
),
unique_players as (
  select
    date_trunc('day', m.start_time) as date_bucket,
    m.stats_bucket_id,
    m.map_id,
    count(distinct mp.steam_id)::bigint as unique_players
  from
    match m
    join match_players mp using (match_id)
  group by
    date_bucket,
    m.stats_bucket_id,
    m.map_id
)
select
  mm.date_bucket,
  mm.stats_bucket_id,
  mm.map_id,
  mm.matches_played,
  mm.matches_decided,
  mm.matches_drawn,
  mm.red_wins,
  mm.blu_wins,
  mm.total_duration_ms,
  mm.rounds_played,
  mm.stalemate_rounds,
  mm.player_appearances,
  coalesce(up.unique_players, 0)::bigint as unique_players,
  mm.kills,
  mm.assists,
  mm.deaths,
  mm.damage,
  mm.healing,
  mm.captures,
  mm.captures_blocked,
  mm.first_played_at,
  mm.last_played_at
from
  map_metrics mm
  left join unique_players up using (date_bucket, stats_bucket_id, map_id);

create unique index stats_map_daily_view_uindex on stats_map_daily_view (date_bucket, stats_bucket_id, map_id);
create index stats_map_daily_bucket_date_idx on stats_map_daily_view (stats_bucket_id, date_bucket);
create index stats_map_daily_map_idx on stats_map_daily_view (map_id);

-- One row per player who appeared on a map on a given day.
create materialized view stats_map_daily_player_view as
select distinct
  date_trunc('day', m.start_time) as date_bucket,
  m.stats_bucket_id,
  m.map_id,
  p.steam_id
from
  match m
  join match_round r on r.match_id = m.match_id
  join match_round_player p using (round_id);

create unique index stats_map_daily_player_view_uindex on stats_map_daily_player_view (date_bucket, stats_bucket_id, map_id, steam_id);
create index stats_map_daily_player_range_idx on stats_map_daily_player_view (stats_bucket_id, map_id, date_bucket);
create index stats_map_daily_player_date_idx on stats_map_daily_player_view (stats_bucket_id, date_bucket);

create materialized view stats_map_alltime_view as
with match_players as (
  select
    r.match_id,
    p.steam_id,
    sum(p.kills)::bigint as kills,
    sum(p.assists)::bigint as assists,
    sum(p.deaths)::bigint as deaths,
    sum(p.damage)::bigint as damage,
    sum(p.healing)::bigint as healing,
    sum(p.captures)::bigint as captures,
    sum(p.captures_blocked)::bigint as captures_blocked
  from
    match_round r
    join match_round_player p using (round_id)
  group by
    r.match_id,
    p.steam_id
),
match_rounds as (
  select
    match_id,
    count(*)::bigint as rounds_played,
    count(*) filter (where is_stalemate)::bigint as stalemate_rounds
  from
    match_round
  group by
    match_id
),
match_player_totals as (
  select
    match_id,
    sum(kills)::bigint as kills,
    sum(assists)::bigint as assists,
    sum(deaths)::bigint as deaths,
    sum(damage)::bigint as damage,
    sum(healing)::bigint as healing,
    sum(captures)::bigint as captures,
    sum(captures_blocked)::bigint as captures_blocked,
    count(*)::bigint as player_appearances
  from
    match_players
  group by
    match_id
),
map_metrics as (
  select
    m.stats_bucket_id,
    m.map_id,
    count(*)::bigint as matches_played,
    count(*) filter (where m.score_red <> m.score_blu)::bigint as matches_decided,
    count(*) filter (where m.score_red = m.score_blu)::bigint as matches_drawn,
    count(*) filter (where m.score_red > m.score_blu)::bigint as red_wins,
    count(*) filter (where m.score_blu > m.score_red)::bigint as blu_wins,
    sum(m.duration_ms)::bigint as total_duration_ms,
    coalesce(sum(mr.rounds_played), 0)::bigint as rounds_played,
    coalesce(sum(mr.stalemate_rounds), 0)::bigint as stalemate_rounds,
    coalesce(sum(mpt.player_appearances), 0)::bigint as player_appearances,
    coalesce(sum(mpt.kills), 0)::bigint as kills,
    coalesce(sum(mpt.assists), 0)::bigint as assists,
    coalesce(sum(mpt.deaths), 0)::bigint as deaths,
    coalesce(sum(mpt.damage), 0)::bigint as damage,
    coalesce(sum(mpt.healing), 0)::bigint as healing,
    coalesce(sum(mpt.captures), 0)::bigint as captures,
    coalesce(sum(mpt.captures_blocked), 0)::bigint as captures_blocked,
    min(m.start_time) as first_played_at,
    max(m.start_time) as last_played_at
  from
    match m
    left join match_rounds mr using (match_id)
    left join match_player_totals mpt using (match_id)
  group by
    m.stats_bucket_id,
    m.map_id
),
unique_players as (
  select
    m.stats_bucket_id,
    m.map_id,
    count(distinct mp.steam_id)::bigint as unique_players
  from
    match m
    join match_players mp using (match_id)
  group by
    m.stats_bucket_id,
    m.map_id
)
select
  mm.stats_bucket_id,
  mm.map_id,
  mm.matches_played,
  mm.matches_decided,
  mm.matches_drawn,
  mm.red_wins,
  mm.blu_wins,
  mm.total_duration_ms,
  mm.rounds_played,
  mm.stalemate_rounds,
  mm.player_appearances,
  coalesce(up.unique_players, 0)::bigint as unique_players,
  mm.kills,
  mm.assists,
  mm.deaths,
  mm.damage,
  mm.healing,
  mm.captures,
  mm.captures_blocked,
  mm.first_played_at,
  mm.last_played_at
from
  map_metrics mm
  left join unique_players up using (stats_bucket_id, map_id);

create unique index stats_map_alltime_view_uindex on stats_map_alltime_view (stats_bucket_id, map_id);
create index stats_map_alltime_map_idx on stats_map_alltime_view (map_id);

refresh materialized view stats_map_daily_view;
refresh materialized view stats_map_daily_player_view;
refresh materialized view stats_map_alltime_view;

commit;
