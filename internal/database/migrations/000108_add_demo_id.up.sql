BEGIN;

ALTER TABLE match
    ADD COLUMN demo_id int4 REFERENCES demo (demo_id);

CREATE INDEX IF NOT EXISTS match_demo_id ON match (demo_id);

COMMIT;
