BEGIN;

drop table if exists filtered_word;

create table if not exists filtered_word
(
    filter_id  bigserial primary key,
    author_id  bigint    not null
        constraint filtered_word_author_id_fk
            references person,
    pattern    text      not null,
    is_regex   boolean   not null default false,
    is_enabled boolean not null default true,
    trigger_count bigint not null default 0,
    created_on timestamp not null,
    updated_on  timestamp not null
);

create unique index if not exists filtered_word_pattern_uindex
    on filtered_word (pattern);

COMMIT;
