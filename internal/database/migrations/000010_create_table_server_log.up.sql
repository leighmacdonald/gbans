begin;

create table if not exists server_log
(
    log_id     bigserial
        constraint server_log_pk
            primary key,
    server_id  int              not null,
    event_type int    default 0 not null,
    payload    jsonb            not null,
    source_id  bigint default 0 not null,
    target_id  bigint default 0 not null,
    created_on timestamptz      not null,
    CONSTRAINT fk_server_id FOREIGN KEY (server_id) REFERENCES server (server_id)
);

create index if not exists server_log_server_id_idx on server_log (server_id);

commit;
