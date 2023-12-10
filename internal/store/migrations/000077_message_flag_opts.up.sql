BEGIN;

ALTER TABLE filtered_word
    ADD COLUMN action integer default 1;

ALTER TABLE filtered_word
    ADD COLUMN duration text default '1w';

COMMIT;
