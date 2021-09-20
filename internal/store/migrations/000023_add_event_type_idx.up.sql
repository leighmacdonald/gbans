begin;

create index if not exists server_log_event_type_index
    on server_log (event_type);

commit;
