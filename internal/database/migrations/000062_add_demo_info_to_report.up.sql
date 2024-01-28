BEGIN;

alter table if exists report add column demo_name text not null default '';
alter table if exists report add column demo_tick int not null default -1;

COMMIT;
