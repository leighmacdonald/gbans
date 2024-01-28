begin;

create table if not exists demo
(
    demo_id serial constraint demo_pk primary key,
    server_id int not null
        constraint demo_server_server_id_fk references server on update cascade on delete cascade,
    title varchar not null,
    raw_data bytea not null,
    size bigint not null,
    downloads int default 0 not null
);

create unique index demo_title_uindex
    on demo (title);

alter table server
    add region varchar default 'us' not null;

alter table server
    add cc varchar(2) default 'us' not null;

alter table server
    add location geography default ST_GeomFromText('POINT(10.1 54.0)', 4326) not null;

create index ban_created_on_index
    on ban (created_on);

commit;
