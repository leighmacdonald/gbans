-- Matches with no round player data previously produced a group row with a
-- NULL steam_id and NULL aggregates in the summary views (left joins over
-- empty match_round_player(_variants) tables), which cannot be scanned into
-- the non-nullable stat fields. Join player data with inner joins so such
-- matches are excluded from the summaries.

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
  sum(p.ignites) as ignites,
  sum(p.heals) as heals,
  sum(p.healed) as healed,
  sum(p.crossbow_heals) as crossbow_heals,
  sum(p.crossbow_healing) as crossbow_healing,
  sum(p.heal_on_hit) as heal_on_hit,
  sum(p.building_healing) as building_healing,
  sum(p.dropped_ubers) as dropped_ubers,
  sum(p.reflects) as reflects,
  sum(p.defenses) as defenses,
  sum(p.direct_hits) as direct_hits,
  sum(p.teleports) as teleports,
  sum(p.push_distance) as push_distance,
  sum(p.environmental_deaths) as environmental_deaths,
  sum(p.environmental_kills) as environmental_kills,
  sum(p.object_placed) as object_placed,
  sum(p.object_upgraded) as object_upgraded,
  sum(p.object_carried) as object_carried,
  sum(p.object_dropped) as object_dropped,
  sum(p.object_removed) as object_removed,
  sum(p.object_detonated) as object_detonated,
  sum(p.ammo_packs) as ammo_packs,
  sum(p.health_packs) as health_packs,
  sum(p.health_pack_healing) as health_pack_healing
from
  match m
  join match_round r USING (match_id)
  join match_round_player p USING (round_id)
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
  sum(v.charges_quickfix) as charges_quickfix,
  sum(v.shots) as shots,
  sum(v.hits) as hits,
  sum(v.objects_built) as objects_built,
  sum(v.objects_destroyed) as objects_destroyed,
  sum(v.heals) as heals,
  sum(v.healed) as healed,
  sum(v.crossbow_heals) as crossbow_heals,
  sum(v.crossbow_healing) as crossbow_healing,
  sum(v.heal_on_hit) as heal_on_hit,
  sum(v.extinguishes) as extinguishes,
  sum(v.building_healing) as building_healing,
  sum(v.dropped_ubers) as dropped_ubers,
  sum(v.reflects) as reflects,
  sum(v.defenses) as defenses,
  sum(v.direct_hits) as direct_hits,
  sum(v.teleports) as teleports,
  sum(v.push_distance) as push_distance,
  sum(v.environmental_deaths) as environmental_deaths,
  sum(v.environmental_kills) as environmental_kills,
  sum(v.object_placed) as object_placed,
  sum(v.object_upgraded) as object_upgraded,
  sum(v.object_carried) as object_carried,
  sum(v.object_dropped) as object_dropped,
  sum(v.object_removed) as object_removed,
  sum(v.object_detonated) as object_detonated,
  sum(v.ammo_packs) as ammo_packs,
  sum(v.health_packs) as health_packs,
  sum(v.health_pack_healing) as health_pack_healing
from
  match m
  join match_round r using (match_id)
  join match_round_player_variants v using (round_id)
group by
  date_bucket,
  m.stats_bucket_id,
  v.steam_id,
  v.variant;

create index if not exists stats_summary_daily_variant_idx on stats_summary_daily_variants_view (variant);

create index if not exists stats_summary_daily_variant_steamid_idx on stats_summary_daily_variants_view (steam_id);

create index if not exists stats_summary_daily_overall_steamid_idx on stats_summary_daily_overall_view (steam_id);

refresh materialized view stats_summary_daily_overall_view;

refresh materialized view stats_summary_daily_variants_view;
