begin;

alter table if exists server add name text default '' not null;

commit;
