DROP INDEX IF EXISTS demo_title_uindex;

ALTER TABLE ban
DROP COLUMN IF EXISTS demo_id;

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS anticheat_id bigint references anticheat (anticheat_id);
