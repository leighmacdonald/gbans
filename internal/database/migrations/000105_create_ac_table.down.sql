BEGIN;

DROP TABLE IF EXISTS anticheat;
DROP TYPE IF EXISTS detection_type;

ALTER TABLE config
    DROP COLUMN IF EXISTS ssh_stac_path_fmt;


COMMIT;
