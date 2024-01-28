begin;

alter table if exists server drop column if exists latitude;
alter table if exists server drop column if exists longitude;

alter table if exists server add column if not exists location geography;
commit;
