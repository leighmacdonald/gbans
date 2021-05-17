begin;

alter table person add permission_level int8 default 0 not null;

commit;
