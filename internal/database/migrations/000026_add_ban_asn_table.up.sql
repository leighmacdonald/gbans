begin;

create table if not exists ban_asn
(
    ban_asn_id  serial
        constraint ban_asn_pk primary key,
    as_num      bigint           not null,
    origin      int    default 0 not null,
    author_id   bigint           not null
        constraint ban_asn_person_steam_id_fk references person,
    target_id   bigint default 0 not null,
    reason      varchar,
    valid_until timestamptz      not null,
    created_on  timestamptz      not null,
    updated_on  timestamptz      not null
);

create unique index if not exists ban_asn_as_num_uindex on ban_asn (as_num);

commit;
