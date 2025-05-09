BEGIN;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_sdr_enabled;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_sdr_dns_enabled;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_cf_key;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_cf_email;

COMMIT;
