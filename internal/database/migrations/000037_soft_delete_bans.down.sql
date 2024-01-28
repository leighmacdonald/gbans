begin;

alter table if exists ban
    drop column if exists deleted;
alter table if exists ban_asn
    drop column if exists deleted;
alter table if exists ban_net
    drop column if exists deleted;


commit;
