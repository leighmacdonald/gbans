BEGIN;

ALTER TABLE config ADD COLUMN IF NOT EXISTS demo_parser_url text default 'http://localhost:8811/';

COMMIT;
