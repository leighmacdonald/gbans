BEGIN;

DROP TABLE IF EXISTS playerqueue_messages;

ALTER TABLE config
    DROP COLUMN IF EXISTS general_playerqueue_enabled;

COMMIT;
