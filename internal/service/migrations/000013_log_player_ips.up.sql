begin;

alter table person DROP ip_addr;
alter table person_ip RENAME address TO ip_addr;

commit;
