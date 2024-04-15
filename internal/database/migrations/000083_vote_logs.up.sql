BEGIN;

create table vote_result
(
    vote_id    serial primary key,
    server_id  int       not null references server (server_id),
    match_id   uuid      not null references match (match_id),
    source_id  bigint    not null references person (steam_id),
    target_id  bigint references person (steam_id),
    passed     bool      not null,
    name       text      not null,
    created_on timestamp not null
);

COMMIT;
