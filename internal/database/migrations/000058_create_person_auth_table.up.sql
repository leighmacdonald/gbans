BEGIN;

CREATE TABLE person_auth
(
    person_auth_id bigserial primary key,
    steam_id       bigint
        constraint report_reported_steam_id_fk
            references person
            on update cascade on delete cascade,
    ip_addr        inet        not null,
    refresh_token  text        not null,
    created_on     timestamptz not null
);

create unique index person_auth_uindex
    on person_auth (steam_id, ip_addr);

alter table if exists server drop column if exists token;

COMMIT;
