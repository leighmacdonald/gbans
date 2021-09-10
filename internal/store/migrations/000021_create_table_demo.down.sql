begin;

DROP TABLE IF EXISTS demo;

ALTER TABLE IF EXISTS server DROP COLUMN IF EXISTS region;
ALTER TABLE IF EXISTS server DROP COLUMN IF EXISTS cc;
ALTER TABLE IF EXISTS server DROP COLUMN IF EXISTS location;

drop index if exists ban_created_on_index;

commit;
