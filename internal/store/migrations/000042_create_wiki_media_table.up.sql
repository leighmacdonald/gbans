begin;

create table if not exists wiki_media
(
    wiki_media_id serial
        constraint wiki_media_pk
            primary key,
    author_id     bigint                   not null
        constraint wiki_media_author_id_fk
            references person
            on update cascade on delete restrict,
    mime_type     varchar                  not null,
    contents      bytea                    not null,
    name          varchar                  not null,
    size          bigint                   not null,
    deleted       boolean default false    not null,
    created_on    timestamp with time zone not null,
    updated_on    timestamp with time zone not null
);

create unique index if not exists wiki_media_name_uindex on wiki_media (name);

commit;
