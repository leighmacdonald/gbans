begin;

alter table if exists filtered_word drop column if exists discord_id;
alter table if exists filtered_word drop column if exists discord_created_on;
alter table if exists filtered_word drop column if exists filter_name;

drop index if exists filter_name_uindex;
create unique index if not exists filtered_word_word_uindex
    on filtered_word (word);

commit;
