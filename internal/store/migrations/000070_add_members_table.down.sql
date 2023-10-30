BEGIN;

drop table if exists members;

ALTER TABLE IF EXISTS ban
    DROP COLUMN IF EXISTS include_friends;

COMMIT;
