begin;

drop table if exists anonmod_identity;

alter table if exists config
    drop column if exists general_anonmod_enabled;

commit;
