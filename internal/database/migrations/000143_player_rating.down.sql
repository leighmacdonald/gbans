begin;

alter table if exists config
    drop column if exists stats_rating_algorithm;

drop materialized view if exists stats_player_rating_view;
drop table if exists player_rating_history;
drop table if exists player_rating;

commit;
