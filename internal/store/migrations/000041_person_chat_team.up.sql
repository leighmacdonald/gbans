begin;

create table if not exists person_chat
(
    person_chat_id serial
        constraint person_chat_pk
            primary key,
    steam_id       bigint                not null
        constraint person_chat_person_steam_id_fk
            references person,
    server_id      integer               not null
        constraint person_chat_server_server_id_fk
            references server,
    body           text                  not null,
    created_on     timestamp             not null,
    teamchat       boolean default false not null
);

create index if not exists match_player_steam_id_index on match_player (steam_id);

commit;
