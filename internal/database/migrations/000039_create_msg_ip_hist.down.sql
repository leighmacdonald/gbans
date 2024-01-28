begin;

drop table if exists person_chat;

alter table person_ip drop column if exists server_id;

create table server_log
(
    log_id            bigserial
        constraint server_log_pk
            primary key,
    server_id         integer                  not null
        constraint fk_server_id
            references server,
    event_type        integer  default 0       not null,
    source_id         bigint   default 0       not null,
    target_id         bigint   default 0       not null,
    created_on        timestamp with time zone not null,
    weapon            smallint,
    damage            smallint,
    attacker_position geometry(PointZ),
    victim_position   geometry(PointZ),
    assister_position geometry(PointZ),
    item              smallint,
    player_class      integer  default 0,
    player_team       smallint default 0       not null,
    meta_data         jsonb,
    healing           bigint   default 0       not null
);

alter table server_log
    owner to gbans;



commit;
