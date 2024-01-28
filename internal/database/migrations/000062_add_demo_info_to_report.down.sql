BEGIN;

alter table if exists report drop column if exists demo_name;
alter table if exists report drop column if exists demo_tick;

COMMIT;
