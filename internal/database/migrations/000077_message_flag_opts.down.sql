BEGIN;


ALTER TABLE filtered_word
    DROP COLUMN action;

ALTER TABLE filtered_word
    DROP COLUMN duration;

COMMIT;

