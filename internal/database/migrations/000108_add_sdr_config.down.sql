BEGIN;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_sdr_enabled;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_sdr_dns_enabled;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_cf_key;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_cf_email;
ALTER TABLE config
    DROP COLUMN IF EXISTS network_cf_zone_id;

ALTER TABLE server DROP COLUMN IF EXISTS address_sdr;

COMMIT;
