DROP INDEX IF EXISTS demo_title_uindex;

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS anticheat_id bigint references anticheat (anticheat_id);

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS demo_id integer references demo (demo_id);

ALTER TABLE person_messages
ADD COLUMN IF NOT EXISTS demo_id integer references demo (demo_id);

ALTER TABLE person_messages
ADD COLUMN IF NOT EXISTS demo_tick integer;
