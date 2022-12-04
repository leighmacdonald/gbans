BEGIN;

create table if not exists person_notification
(
    person_notification_id bigserial,
    steam_id                bigint                   not null,
    read                    boolean default false    not null,
    deleted                 boolean default false    not null,
    severity                integer                  not null,
    message                 text                     not null,
    link                    text                     not null default '',
    count                   integer default 0        not null,
    created_on              timestamp with time zone not null
);

alter table person_notification add constraint person_notification_steam_id_fk
    foreign key (steam_id) references person;

COMMIT;
