BEGIN;

ALTER TABLE server
    ADD COLUMN address_internal text not null default '';

COMMIT;
