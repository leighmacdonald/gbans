drop materialized view if exists stats_weapons_view;

drop materialized view if exists stats_summary_daily_overall_view;

drop materialized view if exists stats_summary_daily_variants_view;

drop table if exists match_round_player_variants;

drop table if exists match_round_player;

drop table if exists match_round;

drop table if exists match;

alter table server
drop column if exists stats_bucket_id;

drop table if exists stats_bucket;

drop type if exists player_class;

drop type if exists player_team;

ALTER TABLE demo ADD COLUMN deleted boolean NOT NULL DEFAULT false;

ALTER TABLE asset DROP COLUMN deleted;

ALTER TABLE server DROP COLUMN IF EXISTS discord_seed_channel_id;
