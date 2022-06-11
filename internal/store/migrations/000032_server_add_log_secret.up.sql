begin;

alter table if exists server add column log_secret int default 0;

commit;