begin;

alter table if exists net_proxy alter column as_num type bigint using as_num::bigint;
alter table if exists net_asn alter column as_num type bigint using as_num::bigint;

commit;
