BEGIN;

ALTER TABLE config
    DROP COLUMN IF EXISTS demo_parser_url;

COMMIT;
