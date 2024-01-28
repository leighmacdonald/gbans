BEGIN;

ALTER TABLE filtered_word ADD COLUMN weight int default 1;

COMMIT;
