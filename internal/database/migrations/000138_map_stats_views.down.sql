begin;

drop materialized view if exists stats_map_alltime_view;
drop materialized view if exists stats_map_daily_player_view;
drop materialized view if exists stats_map_daily_view;

commit;
