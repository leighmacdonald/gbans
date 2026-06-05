CREATE UNIQUE INDEX IF NOT EXISTS demo_title_uindex ON demos (title);

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS demo_id int references demo (demo_id);

ALTER TABLE ban
DROP COLUMN IF EXISTS anticheat_id;
