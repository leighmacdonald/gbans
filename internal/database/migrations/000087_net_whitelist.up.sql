BEGIN;

create table person_whitelist
(
    steam_id bigint primary key references person (steam_id) ON DELETE CASCADE,
    created_on timestamp not null,
    updated_on timestamp not null
);
COMMIT;
