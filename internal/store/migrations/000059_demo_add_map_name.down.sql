BEGIN;

alter table if exists demo drop column if exists map_name;
alter table if exists demo drop column if exists created_on;
alter table if exists demo drop column if exists archive;
alter table if exists demo drop column if exists stats;

COMMIT;
