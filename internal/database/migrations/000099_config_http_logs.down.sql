BEGIN;

ALTER TABLE config DROP COLUMN logging_http_enabled;
ALTER TABLE config DROP COLUMN logging_http_otel_enabled;
ALTER TABLE config DROP COLUMN logging_http_level;
COMMIT;
