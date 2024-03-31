BEGIN;

-- Remove our custom iprange type, ip4r has its own
ALTER TABLE net_asn DROP column ip_range;
ALTER TABLE net_location DROP column ip_range;
ALTER TABLE net_proxy DROP column ip_range;

DROP TYPE iprange;

-- Install extension
CREATE EXTENSION ip4r;

-- Clean out the tables since we dont have to worry about these and can just re-import.
TRUNCATE net_asn;
TRUNCATE net_location;
TRUNCATE net_proxy;

-- Replace ip_range with the ip4r  type
ALTER TABLE net_asn ADD COLUMN ip_range iprange not null;
ALTER TABLE net_location ADD COLUMN ip_range iprange not null;
ALTER TABLE net_proxy ADD COLUMN ip_range iprange not null;

-- Add a temp column to map to the new type
ALTER TABLE person_connections ADD COLUMN new_ip4 ip4;

-- Load our new data type, its using "dual"/128bit format, so we just trim the prefix as we
-- don't actually care about ipv6 since its not supported anyways.
UPDATE person_connections set new_ip4 = TRIM('::ffff:' from abbrev(ip_addr))::ip4 where (1=1);

-- Remove old column
ALTER TABLE person_connections DROP COLUMN ip_addr;

-- Replace with our new column
ALTER TABLE person_connections RENAME COLUMN new_ip4 TO ip_addr;

-- Add index
CREATE INDEX ip_addr_idx ON person_connections (ip_addr);

COMMIT;
