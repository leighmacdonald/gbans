begin;

ALTER TABLE IF EXISTS server ADD COLUMN IF NOT EXISTS is_enabled bool not null default true;

commit;
