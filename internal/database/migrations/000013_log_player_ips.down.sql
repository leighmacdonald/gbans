begin;

alter table person add ip_addr varchar default '' not null;
alter table person_ip RENAME ip_addr TO address;
commit;