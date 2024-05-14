BEGIN;

alter table report drop column demo_id;
alter table report add column demo_name text default '';

COMMIT;
