begin;

ALTER TABLE IF EXISTS server ADD COLUMN IF NOT EXISTS enabled bool not null default true;

commit;
