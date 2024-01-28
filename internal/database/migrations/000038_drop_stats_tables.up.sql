begin;

drop table if exists stats_global_daily;
drop table if exists stats_global_monthly;
drop table if exists stats_global_weekly;

drop table if exists stats_map_daily;
drop table if exists stats_map_monthly;
drop table if exists stats_map_weekly;

drop table if exists stats_player_daily;
drop table if exists stats_player_monthly;
drop table if exists stats_player_weekly;

drop table if exists stats_server_daily;
drop table if exists stats_server_monthly;
drop table if exists stats_server_weekly;

drop index if exists server_log_server_id_idx;
drop index if exists server_log_source_id_idx;
drop index if exists server_log_target_id_idx;
drop index if exists server_log_created_on_idx;
drop index if exists server_log_event_type_index;
drop index if exists md_connected_address_idx;
drop index if exists md_say_msg_idx;
drop index if exists server_log_created_on_desc_idx;

commit;
