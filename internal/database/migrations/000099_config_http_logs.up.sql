BEGIN;

ALTER TABLE config ADD COLUMN logging_http_enabled bool not null default false;
ALTER TABLE config ADD COLUMN logging_http_otel_enabled bool not null default false;
ALTER TABLE config ADD COLUMN logging_http_level text not null default 'warn';

COMMIT;
