begin;

alter table ban
    add deleted bool default false not null;
alter table ban_asn
    add deleted bool default false not null;
alter table ban_net
    add deleted bool default false not null;

commit;
