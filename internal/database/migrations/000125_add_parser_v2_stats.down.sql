drop materialized view if exists stats_summary_daily_overall_view;

drop materialized view if exists stats_summary_daily_variants_view;

drop table if exists match_kill;

alter table match_round_player
  drop column if exists heals,
  drop column if exists healed,
  drop column if exists crossbow_heals,
  drop column if exists crossbow_healing,
  drop column if exists heal_on_hit,
  drop column if exists building_healing,
  drop column if exists dropped_ubers,
  drop column if exists reflects,
  drop column if exists defenses,
  drop column if exists direct_hits,
  drop column if exists teleports,
  drop column if exists push_distance,
  drop column if exists environmental_deaths,
  drop column if exists environmental_kills,
  drop column if exists object_placed,
  drop column if exists object_upgraded,
  drop column if exists object_carried,
  drop column if exists object_dropped,
  drop column if exists object_removed,
  drop column if exists object_detonated,
  drop column if exists ammo_packs,
  drop column if exists health_packs,
  drop column if exists health_pack_healing;

alter table match_round_player_variants
  drop column if exists heals,
  drop column if exists healed,
  drop column if exists crossbow_heals,
  drop column if exists crossbow_healing,
  drop column if exists heal_on_hit,
  drop column if exists extinguishes,
  drop column if exists building_healing,
  drop column if exists dropped_ubers,
  drop column if exists reflects,
  drop column if exists defenses,
  drop column if exists direct_hits,
  drop column if exists teleports,
  drop column if exists push_distance,
  drop column if exists environmental_deaths,
  drop column if exists environmental_kills,
  drop column if exists object_placed,
  drop column if exists object_upgraded,
  drop column if exists object_carried,
  drop column if exists object_dropped,
  drop column if exists object_removed,
  drop column if exists object_detonated,
  drop column if exists ammo_packs,
  drop column if exists health_packs,
  drop column if exists health_pack_healing;

drop materialized view if exists stats_summary_daily_overall_view;

create materialized view stats_summary_daily_overall_view as
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

drop materialized view if exists stats_summary_daily_variants_view;

create materialized view stats_summary_daily_variants_view as
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

refresh materialized view stats_weapons_view;

refresh materialized view stats_summary_daily_overall_view;

refresh materialized view stats_summary_daily_variants_view;
