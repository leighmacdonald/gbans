begin;

ALTER TABLE net_asn DROP COLUMN IF EXISTS ip_range;
ALTER TABLE net_proxy DROP COLUMN IF EXISTS ip_range;
ALTER TABLE net_location DROP COLUMN IF EXISTS ip_range;

DROP TYPE IF EXISTS iprange;

drop function if exists iprange(inet, inet);
drop function if exists iprange(inet, inet, text);

commit;
