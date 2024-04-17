BEGIN;

alter table ban
    add column evade_ok bool not null default false;

COMMIT;
