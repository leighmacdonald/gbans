begin;

alter table if exists server drop column if exists default_map;

commit;
