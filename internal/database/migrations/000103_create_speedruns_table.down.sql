BEGIN;

DROP TABLE IF EXISTS speedrun_rounds_runners;
DROP TABLE IF EXISTS speedrun_round;
DROP TABLE IF EXISTS speedrun_runners;
DROP TABLE IF EXISTS speedrun;
DROP TABLE IF EXISTS map;

ALTER TABLE config
    DROP COLUMN IF EXISTS general_speedruns_enabled;

COMMIT;
