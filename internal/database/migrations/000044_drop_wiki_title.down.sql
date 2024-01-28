begin;

alter table if exists wiki add column title text not null;

alter table if exists media rename column media_id to wiki_media_id;

alter table if exists media rename to wikI_media;

create table report_media
(
    report_media_id serial
        constraint report_media_pk
            primary key,
    report_id       int                   not null
        constraint report_media_report_id_fk
            references report
            on update cascade on delete restrict,
    author_id       bigint                not null
        constraint report_media_author_id_fk
            references person
            on update cascade on delete restrict,
    mime_type       varchar               not null,
    contents        bytea                 not null,
    deleted         boolean default false not null,
    created_on      timestamptz           not null,
    updated_on      timestamptz           not null
);

alter table if exists report add column title text not null;

commit;
