BEGIN;

alter table ban
    drop column evade_ok;

COMMIT;
