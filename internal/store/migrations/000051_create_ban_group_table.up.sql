begin;

create table if not exists ban_group
(
    ban_group_id      bigserial
        constraint ban_group_pk
            primary key,
    source_id         bigint    not null,
    target_id         bigint    not null default 0,
    group_id          bigint    not null,
    group_name        text      not null default '',
    is_enabled        bool               default true not null,
    deleted           bool               default false not null,
    note              text      not null default '',
    unban_reason_text text      not null default '',
    origin            int       not null default 0,
    created_on        timestamp not null,
    updated_on        timestamp not null,
    valid_until       timestamp not null
);

create unique index if not exists ban_group_group_id_uindex on ban_group (group_id);

alter table if exists ban
    rename column steam_id to target_id;
alter table if exists ban
    rename column author_id to source_id;
alter table if exists ban
    rename column ban_source to origin;
alter table if exists ban
    add column if not exists is_enabled bool default true not null;

alter table if exists ban_net
    rename column source to origin;
alter table if exists ban_net
    add column if not exists note text default '' not null;
alter table if exists ban_net
    add column if not exists unban_reason_text text default '' not null;
alter table if exists ban_net
    add column if not exists is_enabled bool default true not null;
alter table if exists ban_net
    add column if not exists source_id bigint default 0 not null;
alter table if exists ban_net
    add column if not exists target_id bigint default 0 not null;

alter table if exists ban_asn
    rename column author_id to source_id;
alter table if exists ban_asn
    add column if not exists unban_reason_text text default '' not null;
alter table if exists ban_asn
    add column if not exists note text default '' not null;
alter table if exists ban_asn
    add column if not exists valid_until timestamp not null;
alter table if exists ban_asn
    add column if not exists is_enabled bool default true not null;

commit;
