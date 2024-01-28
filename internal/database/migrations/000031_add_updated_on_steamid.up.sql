begin;

alter table if exists person add column updated_on_steam timestamp default now();

commit;