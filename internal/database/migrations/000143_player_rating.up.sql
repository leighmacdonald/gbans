begin;

-- Skill calculation selected by the admin Stats settings. The default keeps
-- the existing behaviour for installations upgrading with no explicit choice.
alter table if exists config
    add column if not exists stats_rating_algorithm text not null default 'trueskill';

-- Player skill ratings used for matchmaking and team balancing.
--
-- Ratings are a skill estimate (mu, sigma) of a player's contribution to a
-- team, inferred purely from match win/loss by the configured rating
-- calculator. The public display values are persisted alongside the estimate.
-- One rating is kept per player per stats bucket so game modes do not pollute
-- each other.
--
-- Both tables are rebuilt from scratch by the daily stats refresh job (a
-- chronological replay of all rated matches). They are plain tables because
-- the update is path-dependent and cannot be expressed as an aggregate.

create table if not exists player_rating (
  steam_id             bigint not null,
  stats_bucket_id      integer not null
    references stats_bucket (stats_bucket_id) on delete cascade,
  mu                   double precision not null,
  sigma                double precision not null,
  -- Display values are produced by the configured rating calculator so the
  -- public scale is not tied to the TrueSkill formula used to persist mu/sigma.
  rating               integer not null,
  rating_sigma         integer not null,
  rating_conservative  integer not null,
  matches_played       integer not null,
  rounds_played        integer not null,
  -- Visibility and staleness are decided by the configured calculator.
  visible              boolean not null,
  stale                boolean not null,
  wins                 integer not null,
  losses               integer not null,
  draws                integer not null,
  last_match_id        uuid,
  last_match_at        timestamptz,
  calculated_on        timestamptz not null default now(),
  primary key (steam_id, stats_bucket_id)
);

create index if not exists player_rating_bucket_mu_idx
  on player_rating (stats_bucket_id, mu desc);

-- One row per player per rated match, holding the post-match rating. This is
-- what rating-over-time graphs read without replaying the whole history.
create table if not exists player_rating_history (
  steam_id         bigint not null,
  stats_bucket_id  integer not null
    references stats_bucket (stats_bucket_id) on delete cascade,
  match_id         uuid not null
    references match (match_id) on delete cascade,
  match_start_time timestamptz not null,
  mu               double precision not null,
  sigma            double precision not null,
  expected         double precision not null default 0.5,
  outcome          smallint not null check (outcome in (-1, 0, 1)),
  matches_played   integer not null,
  rounds           integer not null default 0,
  primary key (steam_id, stats_bucket_id, match_id)
);

create index if not exists player_rating_history_player_time_idx
  on player_rating_history (steam_id, stats_bucket_id, match_start_time);
create index if not exists player_rating_history_match_idx
  on player_rating_history (match_id);

-- Current rating per player and bucket. Visibility, staleness and the public
-- scale are all produced by the configured calculator and persisted on the
-- row, so the view does not encode any algorithm-specific policy.
create materialized view stats_player_rating_view as
select
  r.stats_bucket_id,
  r.steam_id,
  coalesce(p.personaname, '') as personaname,
  coalesce(p.avatarhash, '') as avatarhash,
  r.matches_played,
  r.rounds_played,
  r.wins,
  r.losses,
  r.draws,
  r.mu,
  r.sigma,
  r.rating,
  r.rating_sigma,
  r.rating_conservative,
  case when r.visible then rk.rank else 0 end as rank,
  not r.visible as provisional,
  r.stale,
  r.last_match_at,
  r.calculated_on
from
  player_rating r
  left join person p using (steam_id)
  left join (
    select
      stats_bucket_id,
      steam_id,
      rank() over (partition by stats_bucket_id order by mu desc) as rank
    from player_rating
    where visible
  ) rk using (stats_bucket_id, steam_id)
where not exists (
  select 1 from ban b
  where b.target_id = r.steam_id
    and b.ban_type = 2
    and b.deleted = false
    and b.valid_until > now()
);

-- Required for REFRESH MATERIALIZED VIEW CONCURRENTLY.
create unique index if not exists stats_player_rating_view_uindex
  on stats_player_rating_view (stats_bucket_id, steam_id);
create index if not exists stats_player_rating_view_rank_idx
  on stats_player_rating_view (stats_bucket_id, rank);

refresh materialized view stats_player_rating_view;

commit;
