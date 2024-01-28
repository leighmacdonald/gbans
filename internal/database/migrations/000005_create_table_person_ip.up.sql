create table if not exists person_ip
(
    steam_id   bigint      not null
        constraint person_ip_person_steam_id_fk
            references person
            on update cascade on delete cascade,
    address    inet        not null,
    created_on timestamptz not null
);