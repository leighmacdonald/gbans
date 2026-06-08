DROP TABLE IF EXISTS match_player_killstreak;

DROP TABLE IF EXISTS match_player_class;

DROP TABLE IF EXISTS match_weapon;

DROP TABLE IF EXISTS match_medic;

DROP TABLE IF EXISTS match_player;

DROP TABLE IF EXISTS match;

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

create table stats_bucket (
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
  preround_healing integer not null,
  healing integer not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  damage integer not null,
  damage_taken integer not null,
  dominations integer not null,
  dominated integer not null,
  revenges integer not null,
  revenged integer not null,
  airshots integer not null,
  headshots integer not null,
  headshot_kills integer not null,
  backstabs integer not null,
  backstab_kills integer not null,
  was_headshots integer not null,
  was_backstabbed integer not null,
  shots integer not null,
  hits integer not null,
  objects_built integer not null,
  objects_destroyed integer not null,
  scoreboard_kills integer not null,
  scoreboard_assists integer not null,
  suicides integer not null,
  scoreboard_deaths integer not null,
  postround_deaths integer not null,
  captures integer not null,
  captures_blocked integer not null,
  scoreboard_damage integer not null,
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
  damage integer not null,
  damage_taken integer not null,
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
  healing integer not null,
  postround_healing integer not null,
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
  damage integer not null,
  damage_taken integer not null,
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
  healing integer not null,
  postround_healing integer not null,
  drops integer not null,
  near_full_charge_death integer not null,
  charges_uber integer not null,
  charges_kritz integer not null,
  charges_vacc integer not null,
  charges_quickfix integer not null,
  was_backstabbed integer not null,
  primary key (class, round_id, steam_id)
);
