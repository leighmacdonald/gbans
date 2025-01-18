BEGIN;

ALTER TABLE config
    DROP COLUMN IF EXISTS address_internal;

COMMIT;
