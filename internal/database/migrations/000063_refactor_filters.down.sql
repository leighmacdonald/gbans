BEGIN;

drop table if exists filtered_word;

create table if not exists filtered_word
(
    word_id    bigserial primary key,
    word       text,
    created_on timestamp not null,
    filter_name text,
    discord_id text,
    discord_create_on timestamp not null
);

create unique index if not exists filtered_word_word_uindex
    on filtered_word (word);

COMMIT;
