begin;

alter table report add reason int default 1 not null;
alter table report add reason_text text default '' not null;

commit;
