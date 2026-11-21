begin;

-- Optional password used when connecting to a server's SourceTV broadcast.
-- Empty means the server does not require one.
alter table if exists server
    add column if not exists stv_password text not null default '';

commit;
