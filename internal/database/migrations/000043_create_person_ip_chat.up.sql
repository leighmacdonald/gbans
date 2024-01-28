begin;

create table if not exists person_connections
(
    person_connection_id bigserial
        primary key,
    steam_id             bigint
        constraint fk_steam_id
            references person,
    ip_addr              inet      not null,
    persona_name          text      not null,
    created_on           timestamp not null
);

create table if not exists person_messages
(
    person_message_id bigserial
        primary key,
    steam_id             bigint
        constraint fk_steam_id
            references person,
    server_id            bigint
        constraint fk_server_id
            references server,
    body                 text      not null,
    persona_name          text      not null,
    team                 boolean   not null,
    created_on           timestamp not null
);

drop table if exists person_names;
drop table if exists person_chat;
drop table if exists person_ip;

commit;
