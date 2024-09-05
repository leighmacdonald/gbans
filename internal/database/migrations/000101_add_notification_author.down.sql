BEGIN;

ALTER TABLE person_notification DROP COLUMN IF EXISTS author_id;

COMMIT;
