BEGIN;

ALTER TABLE demo
    ADD COLUMN raw_data bytea not null;
ALTER TABLE demo
    ADD COLUMN size bigint not null;

COMMIT;
