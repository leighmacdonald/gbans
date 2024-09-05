BEGIN;

ALTER TABLE person_notification ADD COLUMN IF NOT EXISTS author_id bigint REFERENCES person (steam_id);

COMMIT;
