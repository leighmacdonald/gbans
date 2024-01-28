begin;

drop table if exists server_log;

alter table person_ip add server_id int not null;

create table if not exists person_chat
(
    person_chat_id serial
        constraint person_chat_pk
            primary key,
    steam_id       bigint    not null
        constraint person_chat_person_steam_id_fk
            references person,
    server_id      int       not null
        constraint person_chat_server_server_id_fk
            references server,
    body           text      not null,
    created_on     timestamp not null
);


commit;
