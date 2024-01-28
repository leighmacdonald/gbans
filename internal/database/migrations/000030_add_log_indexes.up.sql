begin;

create extension if not exists btree_gist;

-- Moved to in meta_data.msg
alter table if exists server_log drop column if exists extra;


-- Connected event index
delete from server_log where meta_data->>'address' LIKE '%:%';
create index if not exists md_connected_address_idx ON server_log USING BTREE (((meta_data->>'address')::inet)) WHERE event_type = 1004;

-- Enable gist

create index if not exists md_say_msg_idx ON server_log USING gist (((meta_data->>'msg'))) WHERE event_type in (10, 11);

create index if not exists server_log_created_on_desc_idx ON server_log (created_on desc);

commit;