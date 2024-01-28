begin;

drop table if exists ban_appeal;

create table ban_appeal
(
    ban_message_id serial
        constraint ban_message_pk
            primary key,
    ban_id         integer                  not null
        constraint ban_message_ban_id_fk
            references ban
            on update cascade on delete restrict,
    author_id         bigint                   not null
        constraint report_message_author_id_fk
            references person
            on update cascade on delete restrict,
    message_md        text                     not null,
    deleted           boolean default false    not null,
    created_on        timestamp with time zone not null,
    updated_on        timestamp with time zone not null
);

commit;
