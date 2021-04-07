begin;

create table if not exists server
(
    server_id        serial primary key,
    short_name       varchar(32)            not null,
    token            varchar(40) default '' not null,
    address          varchar(128)           not null,
    port             int                    not null,
    rcon             varchar(128)           not null,
    token_created_on timestamp,
    reserved_slots   smallint               not null,
    created_on       timestamp              not null,
    updated_on       timestamp              not null,
    password         varchar(20)            not null
);

create unique index if not exists server_name_uindex
    on server (short_name);

commit;