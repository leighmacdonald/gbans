begin;
create table if not exists ban_net
(
    net_id      bigserial primary key,
    cidr        cidr            not null,
    source      int  default 0  not null,
    created_on  timestamp       not null,
    updated_on  timestamp       not null,
    reason      text default '' not null,
    valid_until timestamp       not null
);
create unique index if not exists ban_net_cidr_uindex
    on ban_net (cidr);
commit;