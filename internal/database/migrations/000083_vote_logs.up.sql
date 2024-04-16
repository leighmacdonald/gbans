BEGIN;

create table vote_result
(
    vote_id    serial primary key,
    server_id  int       not null references server (server_id),
    source_id  bigint    not null references person (steam_id),
    target_id  bigint references person (steam_id),
    success    bool      not null,
    name       text      not null default '',
    code       int       not null default 0,
    created_on timestamp not null
);

COMMIT;
