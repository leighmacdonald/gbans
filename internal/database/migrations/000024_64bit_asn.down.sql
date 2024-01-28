begin;

alter table if exists net_proxy alter column as_num type int using as_num::int;
alter table if exists net_asn alter column as_num type int using as_num::int;

commit;
