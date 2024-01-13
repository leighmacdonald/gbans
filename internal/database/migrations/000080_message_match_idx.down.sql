BEGIN;

DROP INDEX person_messages_match_id_index;

ALTER TABLE match DROP COLUMN demo_name;

COMMIT;
