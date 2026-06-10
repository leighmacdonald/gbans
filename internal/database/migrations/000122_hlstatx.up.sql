DROP TABLE IF EXISTS match_player_killstreak;

DROP TABLE IF EXISTS match_player_class;

DROP TABLE IF EXISTS match_weapon;

DROP TABLE IF EXISTS match_medic;

DROP TABLE IF EXISTS match_player;

DROP TABLE IF EXISTS match;

drop table if exists stats_player_alltime;

drop table if exists stats_demo_player;

drop table if exists stats_demo;

drop table if exists stats_maps;

drop table if exists player_class;

CREATE TYPE player_class AS ENUM(
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

CREATE TYPE player_team AS ENUM('unassigned', 'spec', 'red', 'blu');

create table if not exists stats_bucket (
  stats_bucket_id serial not null primary key,
  bucket_name text not null unique check (LENGTH(bucket_name) > 0)
);

insert into
  stats_bucket (bucket_name)
values
  ('default');

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
  points integer not null,
  connection_count integer not null,
  bonus_points integer not null,
  kills integer not null,
  assists integer not null,
  deaths integer not null,
  postround_kills integer not null,
  postround_assists integer not null,
  preround_healing bigint not null,
  healing bigint not null,
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
  was_headshot integer not null,
  was_backstabbed integer not null,
  shots bigint not null,
  hits bigint not null,
  objects_built integer not null,
  objects_destroyed integer not null,
  scoreboard_kills integer not null,
  scoreboard_assists integer not null,
  suicides integer not null,
  scoreboard_deaths integer not null,
  postround_deaths integer not null,
  captures integer not null,
  captures_blocked integer not null,
  scoreboard_damage bigint not null,
  extinguishes integer not null,
  ignites integer not null,
  buildings_built integer not null,
  buildings_destroyed integer not null,
  primary key (round_id, steam_id)
);

create table match_round_player_weapon (
  weapon text not null check (length(weapon) > 0),
  round_id integer not null references match_round (round_id) on delete cascade,
  steam_id bigint not null references person (steam_id),
  kills integer not null,
  assists integer not null,
  deaths integer not null,
  postround_kills integer not null,
  postround_assists integer not null,
  postround_deaths integer not null,
  damage bigint not null,
  damage_taken bigint not null,
  dominations integer not null,
  dominated integer not null,
  revenges integer not null,
  revenged integer not null,
  airshots integer not null,
  headshot_kills integer not null,
  backstab_kills integer not null,
  headshots integer not null,
  backstabs integer not null,
  was_headshot integer not null,
  was_backstabbed integer not null,
  preround_healing integer not null,
  healing bigint not null,
  postround_healing bigint not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  primary key (weapon, round_id, steam_id)
);

create table match_round_player_class (
  class player_class not null,
  round_id int not null references match_round (round_id) on delete cascade,
  steam_id bigint not null references person (steam_id),
  kills integer not null,
  assists integer not null,
  deaths integer not null,
  postround_kills integer not null,
  postround_assists integer not null,
  postround_deaths integer not null,
  damage bigint not null,
  damage_taken bigint not null,
  dominations integer not null,
  dominated integer not null,
  revenges integer not null,
  revenged integer not null,
  airshots integer not null,
  headshot_kills integer not null,
  backstab_kills integer not null,
  headshots integer not null,
  backstabs integer not null,
  was_headshot integer not null,
  preround_healing integer not null,
  healing bigint not null,
  postround_healing bigint not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  was_backstabbed integer not null,
  primary key (class, round_id, steam_id)
);

create materialized view stats_summary_daily as
select
  date_trunc('day', m.created_on) as date_bucket,
  m.stats_bucket_id,
  p.steam_id,
  SUM(p.points) as point,
  SUM(p.connection_count) as connection_count,
  SUM(p.bonus_points) as bonus_points,
  SUM(p.kills) as kills,
  SUM(p.assists) as assists,
  SUM(p.deaths) as deaths,
  SUM(p.postround_kills) as postround_kills,
  SUM(p.postround_assists) as postround_assists,
  SUM(p.preround_healing) as preround_healing,
  SUM(p.healing) as healing,
  SUM(p.drops) as drops,
  SUM(p.near_full_charge_death) as near_full_charge_death,
  SUM(p.charges_uber) as charges_uber,
  SUM(p.charges_kritz) as charges_kritz,
  SUM(p.charges_vacc) as charges_vacc,
  SUM(p.charges_quickfix) as charges_quickfix,
  SUM(p.damage) as damage,
  SUM(p.damage_taken) as damage_taken,
  SUM(p.dominations) as dominations,
  SUM(p.dominated) as dominated,
  SUM(p.revenges) as revenges,
  SUM(p.revenged) as revenged,
  SUM(p.airshots) as airshots,
  SUM(p.headshots) as headshots,
  SUM(p.headshot_kills) as headshot_kills,
  SUM(p.backstabs) as backstabs,
  SUM(p.backstab_kills) as backstab_kills,
  SUM(p.was_headshot) as was_headshot,
  SUM(p.was_backstabbed) as was_backstabbed,
  SUM(p.shots) as shots,
  SUM(p.hits) as hits,
  SUM(p.objects_built) as objects_built,
  SUM(p.objects_destroyed) as objects_destroyed,
  SUM(p.scoreboard_kills) as scoreboard_kills,
  SUM(p.scoreboard_assists) as scoreboard_assists,
  SUM(p.scoreboard_deaths) as scoreboard_deaths,
  SUM(p.suicides) as suicides,
  SUM(p.postround_deaths) as postround_deaths,
  SUM(p.captures) as captures,
  SUM(p.captures_blocked) as captures_blocked,
  SUM(p.scoreboard_damage) as scoreboard_damage,
  SUM(p.extinguishes) as extinguishes,
  SUM(p.ignites),
  SUM(p.buildings_built) as ignites,
  SUM(p.buildings_destroyed) as buildings_destroyed
from
  match m
  left join match_round r USING (match_id)
  left join match_round_player p USING (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  r.match_id,
  p.steam_id;

create materialized view stats_summary_daily_weapons as
select
  date_trunc('day', m.created_on) as date_bucket,
  m.stats_bucket_id,
  p.steam_id,
  w.weapon,
  SUM(w.kills) as kills,
  SUM(w.assists) as assists,
  SUM(w.deaths) as deaths,
  SUM(w.postround_kills) as postround_kills,
  SUM(w.postround_assists) as postround_assists,
  SUM(w.postround_deaths) as postround_deaths,
  SUM(w.damage) as damage,
  SUM(w.damage_taken) as damage_taken,
  SUM(w.dominations) as dominations,
  SUM(w.dominated) as dominated,
  SUM(w.revenges) as revenges,
  SUM(w.revenged) as revenged,
  SUM(w.airshots) as airshots,
  SUM(w.headshot_kills) as headshot_kills,
  SUM(w.backstab_kills) as backstab_kills,
  SUM(w.headshots) as headshots,
  SUM(w.backstabs) as backstabs,
  SUM(w.was_headshot) as was_headshot,
  SUM(w.was_backstabbed) as was_backstabbed,
  SUM(w.preround_healing) as preround_healing,
  SUM(w.healing) as healing,
  SUM(w.postround_healing) as postround_healing,
  SUM(w.drops) as drops,
  SUM(w.near_full_charge_death) as near_full_charge_death,
  SUM(w.charges_uber) as charges_uber,
  SUM(w.charges_kritz) as charges_kritz,
  SUM(w.charges_vacc) as charges_vacc,
  SUM(w.charges_quickfix) as charges_quickfix
from
  match m
  left join match_round r USING (match_id)
  left join match_round_player p USING (round_id)
  left join match_round_player_weapon w USING (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  r.match_id,
  p.steam_id,
  w.weapon;

create materialized view stats_summary_daily_classes as
select
  date_trunc('day', m.created_on) as date_bucket,
  m.stats_bucket_id,
  p.steam_id,
  c.class,
  SUM(c.kills) as kills,
  SUM(c.assists) as assists,
  SUM(c.deaths) as deaths,
  SUM(c.postround_kills) as postround_kills,
  SUM(c.postround_assists) as postround_assists,
  SUM(c.postround_deaths) as postround_deaths,
  SUM(c.damage) as damage,
  SUM(c.damage_taken) as damage_taken,
  SUM(c.dominations) as dominations,
  SUM(c.dominated) as dominated,
  SUM(c.revenges) as revenges,
  SUM(c.revenged) as revenged,
  SUM(c.airshots) as airshots,
  SUM(c.headshot_kills) as headshot_kills,
  SUM(c.backstab_kills) as backstab_kills,
  SUM(c.headshots) as headshots,
  SUM(c.backstabs) as backstabs,
  SUM(c.was_headshot) as was_headshot,
  SUM(c.was_backstabbed) as was_backstabbed,
  SUM(c.preround_healing) as preround_healing,
  SUM(c.healing) as healing,
  SUM(c.postround_healing) as postround_healing,
  SUM(c.drops) as drops,
  SUM(c.near_full_charge_death) as near_full_charge_death,
  SUM(c.charges_uber) as charges_uber,
  SUM(c.charges_kritz) as charges_kritz,
  SUM(c.charges_vacc) as charges_vacc,
  SUM(c.charges_quickfix) as charges_quickfix
from
  match m
  left join match_round r USING (match_id)
  left join match_round_player p USING (round_id)
  left join match_round_player_class c USING (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  r.match_id,
  p.steam_id,
  c.class;

refresh materialized view stats_summary_daily;

refresh materialized view stats_summary_daily_weapons;

refresh materialized view stats_summary_daily_classes;
