BEGIN;

ALTER TABLE person_connections
    DROP COLUMN server_id;
ALTER TABLE server
    DROP COLUMN enable_stats;

COMMIT;
