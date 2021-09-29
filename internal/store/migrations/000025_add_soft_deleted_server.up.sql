begin;

alter table if exists server add deleted bool default false not null;

commit;
