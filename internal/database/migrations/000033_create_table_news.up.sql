begin;

create table if not exists news
(
    news_id      serial
        constraint news_pk
            primary key,
    title        text              not null,
    body_md      text              not null,
    is_published bool default true not null,
    created_on   timestamp         not null,
    updated_on   timestamp         not null
);

create unique index if not exists news_body_md_uindex on news (body_md);

create unique index if not exists news_title_uindex on news (title);

commit;
