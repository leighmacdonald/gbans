BEGIN;

ALTER TABLE config DROP COLUMN general_srcds_log_addr_external;

COMMIT;
