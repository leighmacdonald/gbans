begin;

ALTER TABLE IF EXISTS server DROP COLUMN IF EXISTS is_enabled;

commit;
