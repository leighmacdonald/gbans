begin;

CREATE TYPE iprange AS range (subtype=inet);

ALTER TABLE net_location ADD COLUMN IF NOT EXISTS ip_range iprange;
UPDATE net_location l SET ip_range = iprange(l.ip_from, l.ip_to);
CREATE INDEX ip_range_idx ON net_location USING gist (ip_range);

ALTER TABLE net_proxy ADD COLUMN IF NOT EXISTS ip_range iprange;
UPDATE net_proxy SET ip_range = iprange(ip_from, ip_to);
CREATE INDEX proxy_ip_range_idx ON net_proxy USING gist (ip_range);

ALTER TABLE net_asn ADD COLUMN IF NOT EXISTS ip_range iprange;
UPDATE net_asn SET ip_range = iprange(ip_from, ip_to);
CREATE INDEX asn_ip_range_idx ON net_asn USING gist (ip_range);

commit;
