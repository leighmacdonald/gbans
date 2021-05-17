begin;

ALTER TABLE net_asn DROP COLUMN IF EXISTS ip_range;
ALTER TABLE net_proxy DROP COLUMN IF EXISTS ip_range;
ALTER TABLE net_location DROP COLUMN IF EXISTS ip_range;

DROP TYPE IF EXISTS iprange;

commit;
