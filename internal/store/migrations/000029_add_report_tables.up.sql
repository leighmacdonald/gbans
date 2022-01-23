begin;

create table report
(
    report_id     serial
        constraint report_pk
            primary key,
    author_id     bigint                not null
        constraint report_author_steam_id_fk
            references person
            on update cascade on delete restrict,
    reported_id   bigint                not null
        constraint report_reported_steam_id_fk
            references person
            on update cascade on delete restrict,
    report_status int     default 0     not null,
    title         text                  not null,
    description   text    default ''    not null,
    deleted       boolean default false not null,
    created_on    timestamptz           not null,
    updated_on    timestamptz           not null
);

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

create table report_message
(
    report_message_id serial
        constraint report_message_pk
            primary key,
    report_id         int                   not null
        constraint report_message_report_id_fk
            references report
            on update cascade on delete restrict,
    author_id         bigint                not null
        constraint report_message_author_id_fk
            references person
            on update cascade on delete restrict,
    message_md        text                  not null,
    deleted           boolean default false not null,
    created_on        timestamptz           not null,
    updated_on        timestamptz           not null
);

commit;