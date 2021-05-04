begin;

alter table person add discord_id varchar default '' not null;

commit;
