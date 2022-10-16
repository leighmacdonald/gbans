BEGIN;

drop table if exists person_auth;

alter table if exists server add column token varchar(40);

COMMIT;
