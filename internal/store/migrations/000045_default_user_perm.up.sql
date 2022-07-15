begin;

alter table if exists person alter column permission_level set default 1;

commit;
