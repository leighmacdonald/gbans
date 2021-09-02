begin;

ALTER TABLE IF EXISTS server DROP COLUMN IF EXISTS enabled;

commit;
