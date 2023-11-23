BEGIN;

ALTER TABLE person_connections ADD COLUMN server_id integer;
ALTER TABLE server ADD COLUMN enable_stats bool default true not null;

COMMIT;
