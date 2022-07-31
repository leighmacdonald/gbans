begin;

alter table if exists filtered_word add filter_name text not null;
alter table if exists filtered_word add discord_id text;
alter table if exists filtered_word add discord_created_on timestamp;

drop index filtered_word_word_uindex;

create unique index filter_name_uindex on filtered_word (filter_name);

commit;
