BEGIN;

create unique index person_auth_uindex
    on person_auth (steam_id, ip_addr);

COMMIT;
