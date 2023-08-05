BEGIN;

DROP TABLE IF EXISTS person_messages_filter;

ALTER TABLE report
    DROP COLUMN person_message_id;

COMMIT;
