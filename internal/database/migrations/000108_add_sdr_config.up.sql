BEGIN;

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS network_sdr_enabled boolean not null DEFAULT false;

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS network_sdr_dns_enabled boolean not null DEFAULT false;

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS network_cf_key text not null DEFAULT '';

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS network_cf_email text not null DEFAULT '';

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS network_cf_zone_id text not null DEFAULT '';

ALTER TABLE server
    ADD COLUMN IF NOT EXISTS sdr_enabled boolean not null default false;

COMMIT;
