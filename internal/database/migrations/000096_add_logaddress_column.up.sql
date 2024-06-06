BEGIN;

ALTER TABLE config ADD COLUMN general_srcds_log_addr_external text not null default '';

COMMIT;
