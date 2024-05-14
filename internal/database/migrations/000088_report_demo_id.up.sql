BEGIN;

alter table report drop column demo_name;

alter table report add column demo_id integer;

COMMIT;
