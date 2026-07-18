DROP TABLE IF EXISTS match_player_killstreak;

DROP TABLE IF EXISTS match_player_class;

DROP TABLE IF EXISTS match_weapon;

DROP TABLE IF EXISTS match_medic;

DROP TABLE IF EXISTS match_player;

DROP TABLE IF EXISTS match_round;

DROP TABLE IF EXISTS match;

drop table if exists stats_player_alltime;

drop table if exists stats_demo_player;

drop table if exists stats_demo;

drop table if exists stats_maps;

drop table if exists player_class;

ALTER TABLE config
DROP COLUMN IF EXISTS network_sdr_dns_enabled;

ALTER TABLE config
DROP COLUMN IF EXISTS network_cf_key;

ALTER TABLE config
DROP COLUMN IF EXISTS network_cf_email;

ALTER TABLE config
DROP COLUMN IF EXISTS network_cf_zone_id;

-- CREATE TYPE player_class AS ENUM(
--   'spectator',
--   'uassigned',
--   'scout',
--   'soldier',
--   'pyro',
--   'demo',
--   'heavy',
--   'engineer',
--   'medic',
--   'sniper',
--   'spy',
--   'saxton'
-- );
CREATE TYPE player_team AS ENUM('unassigned', 'spec', 'red', 'blu');

create table if not exists stats_bucket (
  stats_bucket_id serial not null primary key,
  bucket_name text not null unique check (LENGTH(bucket_name) > 0),
  is_enabled bool not null default true
);

insert into
  stats_bucket (bucket_name)
values
  ('casual'),
  ('1ku'),
  ('mge'),
  ('dodgeball');

alter table server
add column if not exists stats_bucket_id integer references stats_bucket (stats_bucket_id);

CREATE TABLE match (
  match_id uuid not null primary key,
  server_id integer not null references server (server_id) on delete restrict,
  map_id integer not null references map (map_id) on delete restrict,
  demo_id integer not null references demo (demo_id) on delete restrict,
  stats_bucket_id integer not null references stats_bucket (stats_bucket_id) on delete restrict on update cascade default 1,
  hostname text not null,
  score_blu integer not null,
  score_red integer not null,
  start_time timestamptz not null,
  duration_ms bigint not null,
  created_on timestamptz not null
);

create index if not exists match_start_idx on match (start_time);

CREATE TABLE match_round (
  round_id serial not null primary key,
  match_id uuid not null references match (match_id) on delete cascade,
  winner player_team not null,
  is_stalemate boolean not null,
  is_sudden_death boolean not null,
  duration_ms bigint not null
);

CREATE TABLE match_round_player (
  round_id integer not null references match_round (round_id) on delete cascade,
  steam_id bigint not null references person (steam_id),
  team player_team not null,
  mvp bool not null default false,
  tick_start integer not null,
  tick_end integer not null,
  kills integer not null,
  assists integer not null,
  deaths integer not null,
  postround_kills integer not null,
  postround_assists integer not null,
  postround_deaths integer not null,
  preround_healing bigint not null,
  healing bigint not null,
  postround_healing bigint not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  damage bigint not null,
  damage_taken bigint not null,
  dominations integer not null,
  dominated integer not null,
  revenges integer not null,
  revenged integer not null,
  airshots integer not null,
  headshots integer not null,
  headshot_kills integer not null,
  backstabs integer not null,
  backstab_kills integer not null,
  captures integer not null,
  captures_blocked integer not null,
  was_headshot integer not null,
  was_backstabbed integer not null,
  shots bigint not null,
  hits bigint not null,
  objects_built integer not null,
  objects_destroyed integer not null,
  -- Extra player only stuff
  points integer not null,
  connection_count integer not null,
  bonus_points integer not null,
  scoreboard_kills integer not null,
  scoreboard_assists integer not null,
  scoreboard_healing bigint not null,
  scoreboard_deaths integer not null,
  scoreboard_damage bigint not null,
  suicides integer not null,
  extinguishes integer not null,
  ignites integer not null,
  primary key (round_id, steam_id)
);

create table match_round_player_variants (
  variant text not null check (length(variant) > 0),
  round_id integer not null references match_round (round_id) on delete cascade,
  steam_id bigint not null references person (steam_id),
  kills integer not null,
  assists integer not null,
  deaths integer not null,
  postround_kills integer not null,
  postround_assists integer not null,
  postround_deaths integer not null,
  preround_healing integer not null,
  healing bigint not null,
  postround_healing bigint not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  damage bigint not null,
  damage_taken bigint not null,
  dominations integer not null,
  dominated integer not null,
  revenges integer not null,
  revenged integer not null,
  airshots integer not null,
  headshots integer not null,
  headshot_kills integer not null,
  backstabs integer not null,
  backstab_kills integer not null,
  captures integer not null,
  captures_blocked integer not null,
  was_headshot integer not null,
  was_backstabbed integer not null,
  shots bigint not null,
  hits bigint not null,
  objects_built integer not null,
  objects_destroyed integer not null,
  primary key (variant, round_id, steam_id)
);

create materialized view if not exists stats_weapons_view as
select distinct
  variant
from
  match_round_player_variants
where
  variant NOT IN (
    'spectator',
    'uassigned',
    'scout',
    'soldier',
    'pyro',
    'demo',
    'heavy',
    'engineer',
    'medic',
    'sniper',
    'spy',
    'saxton'
  );

create materialized view if not exists stats_summary_daily_overall_view as
select
  date_trunc('day', m.created_on) as date_bucket,
  m.stats_bucket_id,
  rank() over (
    order by
      sum(p.points) desc
  ) as rank,
  p.steam_id,
  sum(p.points) as points,
  sum(p.connection_count) as connection_count,
  sum(p.bonus_points) as bonus_points,
  sum(p.kills) as kills,
  sum(p.assists) as assists,
  sum(p.deaths) as deaths,
  sum(p.postround_kills) as postround_kills,
  sum(p.postround_assists) as postround_assists,
  sum(p.preround_healing) as preround_healing,
  sum(p.healing) as healing,
  sum(p.drops) as drops,
  sum(p.near_full_charge_death) as near_full_charge_death,
  sum(p.charges_uber) as charges_uber,
  sum(p.charges_kritz) as charges_kritz,
  sum(p.charges_vacc) as charges_vacc,
  sum(p.charges_quickfix) as charges_quickfix,
  sum(p.damage) as damage,
  sum(p.damage_taken) as damage_taken,
  sum(p.dominations) as dominations,
  sum(p.dominated) as dominated,
  sum(p.revenges) as revenges,
  sum(p.revenged) as revenged,
  sum(p.airshots) as airshots,
  sum(p.headshots) as headshots,
  sum(p.headshot_kills) as headshot_kills,
  sum(p.backstabs) as backstabs,
  sum(p.backstab_kills) as backstab_kills,
  sum(p.was_headshot) as was_headshot,
  sum(p.was_backstabbed) as was_backstabbed,
  sum(p.shots) as shots,
  sum(p.hits) as hits,
  sum(p.objects_built) as objects_built,
  sum(p.objects_destroyed) as objects_destroyed,
  sum(p.scoreboard_kills) as scoreboard_kills,
  sum(p.scoreboard_assists) as scoreboard_assists,
  sum(p.scoreboard_deaths) as scoreboard_deaths,
  sum(p.suicides) as suicides,
  sum(p.postround_deaths) as postround_deaths,
  sum(p.captures) as captures,
  sum(p.captures_blocked) as captures_blocked,
  sum(p.scoreboard_damage) as scoreboard_damage,
  sum(p.extinguishes) as extinguishes,
  sum(p.ignites) as ignites
from
  match m
  left join match_round r USING (match_id)
  left join match_round_player p USING (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  p.steam_id;

create materialized view if not exists stats_summary_daily_variants_view as
select
  date_trunc('day', m.created_on) as date_bucket,
  m.stats_bucket_id,
  v.steam_id,
  v.variant,
  rank() over (
    partition by
      date_trunc('day', m.created_on),
      v.variant
    order by
      sum(v.kills) desc
  ) as rank,
  sum(v.kills) as kills,
  sum(v.assists) as assists,
  sum(v.deaths) as deaths,
  sum(v.postround_kills) as postround_kills,
  sum(v.postround_assists) as postround_assists,
  sum(v.postround_deaths) as postround_deaths,
  sum(v.damage) as damage,
  sum(v.damage_taken) as damage_taken,
  sum(v.dominations) as dominations,
  sum(v.dominated) as dominated,
  sum(v.revenges) as revenges,
  sum(v.revenged) as revenged,
  sum(v.airshots) as airshots,
  sum(v.headshot_kills) as headshot_kills,
  sum(v.backstab_kills) as backstab_kills,
  sum(v.headshots) as headshots,
  sum(v.backstabs) as backstabs,
  sum(v.was_headshot) as was_headshot,
  sum(v.was_backstabbed) as was_backstabbed,
  sum(v.preround_healing) as preround_healing,
  sum(v.healing) as healing,
  sum(v.postround_healing) as postround_healing,
  sum(v.drops) as drops,
  sum(v.near_full_charge_death) as near_full_charge_death,
  sum(v.charges_uber) as charges_uber,
  sum(v.charges_kritz) as charges_kritz,
  sum(v.charges_vacc) as charges_vacc,
  sum(v.charges_quickfix) as charges_quickfix
from
  match m
  left join match_round r using (match_id)
  left join match_round_player_variants v using (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  v.steam_id,
  v.variant;

create index if not exists stats_summary_daily_variant_idx on stats_summary_daily_variants_view (variant);

create index if not exists stats_summary_daily_variant_steamid_idx on stats_summary_daily_variants_view (steam_id);

create index if not exists stats_summary_daily_overall_steamid_idx on stats_summary_daily_overall_view (steam_id);

-- create index if not exists stats_summary_daily_variant_idx on stats_summary_alltime_variants_view (variant);
-- create index if not exists stats_summary_daily_variant_steamid_idx on stats_summary_alltime_variants_view (steam_id);
refresh materialized view stats_weapons_view;

refresh materialized view stats_summary_daily_overall_view;

refresh materialized view stats_summary_daily_variants_view;

-- refresh materialized view stats_summary_alltime_overall_view;
-- refresh materialized view stats_summary_alltime_variants_view;
--
CREATE INDEX person_messages_server_covering_idx ON person_messages (server_id, person_message_id DESC) INCLUDE (
  steam_id,
  body,
  team,
  created_on,
  persona_name,
  demo_id,
  demo_tick
);

DROP INDEX IF EXISTS idx_created;

CREATE INDEX IF NOT EXISTS idx_created ON person_messages (created_on);

-- This can be used instead of the covering index as well for less disk space, not sure which is better yet.
-- CREATE INDEX person_messages_server_order_idx
-- ON person_messages (server_id, person_message_id DESC);
CREATE INDEX person_messages_steam_order_idx ON person_messages (steam_id, person_message_id DESC);

CREATE INDEX person_messages_filter_message_idx ON person_messages_filter (person_message_id);

CREATE INDEX person_messages_filter_flagged_idx ON person_messages_filter (person_message_id)
WHERE
  person_message_filter_id > 0;

-- Switch to english since we are 99% english content anyways
-- TODO add a note about changing the language as appropriate for others
ALTER TABLE person_messages
DROP COLUMN message_search;

ALTER TABLE person_messages
ADD message_search tsvector GENERATED ALWAYS AS (to_tsvector('english', body)) STORED;

CREATE INDEX idx_message_search ON person_messages USING GIN (message_search);

-- Rebuild name_search with english config
ALTER TABLE person_messages
DROP COLUMN name_search;

ALTER TABLE person_messages
ADD name_search tsvector GENERATED ALWAYS AS (to_tsvector('english', persona_name)) STORED;

CREATE INDEX idx_name_search ON person_messages USING GIN (name_search);

-- Faster to drop and recreate the column than to alter it
alter table person_messages
drop column if exists match_id;

alter table person_messages
add column if not exists match_id uuid REFERENCES match (match_id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_ban_deleted_created ON ban (deleted, created_on DESC);

CREATE INDEX IF NOT EXISTS idx_person_discord ON person (discord_id)
WHERE
  discord_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_person_permission ON person (permission_level);

CREATE INDEX IF NOT EXISTS idx_person_updated_steam ON person (updated_on_steam)
WHERE
  updated_on_steam > '2000-01-01';

CREATE INDEX IF NOT EXISTS idx_msg_thread_created ON forum_message (forum_thread_id, created_on ASC);

alter table report_message
drop constraint report_message_report_id_fk;

alter table report_message
add constraint report_message_report_id_fk foreign key (report_id) references report (report_id) on delete cascade;

alter table person_messages
drop column if exists team;

alter table person_messages
drop constraint if exists person_messages_demo_id_fkey;

alter table person_messages
add constraint person_messages_demo_id_fkey foreign key (demo_id) references demo on delete cascade on update cascade;

ALTER TABLE asset ADD COLUMN deleted boolean NOT NULL DEFAULT false;

ALTER TABLE demo DROP COLUMN deleted;
