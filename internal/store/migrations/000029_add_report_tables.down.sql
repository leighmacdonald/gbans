begin;

DROP TABLE IF EXISTS report_media;
DROP TABLE IF EXISTS report_message;
DROP TABLE IF EXISTS report;

ALTER TABLE IF EXISTS server_log Add COLUMN extra varchar default '';

commit;