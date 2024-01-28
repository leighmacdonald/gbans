begin;

alter table if exists server drop column if exists log_secret;

commit;