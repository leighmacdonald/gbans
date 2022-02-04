begin;

drop table if exists server_log;
drop index if exists md_connected_address_idx;
drop index if exists md_say_msg_idx;
drop index if exists server_log_created_on_desc_idx;
drop extension if exists btree_gist;

commit;