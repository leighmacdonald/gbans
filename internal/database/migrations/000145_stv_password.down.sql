begin;

alter table if exists server
    drop column if exists stv_password;

commit;
