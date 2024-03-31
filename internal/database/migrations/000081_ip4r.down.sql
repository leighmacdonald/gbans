BEGIN;

-- Add a temp column to map to the new type
ALTER TABLE person_connections ADD COLUMN new_inet inet;

-- Load our new data type, its using "dual"/128bit format, so we just trim the prefix as we
-- don't actually care about ipv6 since its not supported anyways.
UPDATE person_connections set new_inet = ip_addr::inet where (1=1);

-- Remove old column
ALTER TABLE person_connections DROP COLUMN ip_addr;

-- Replace with our new column
ALTER TABLE person_connections RENAME COLUMN new_inet TO ip_addr;

-- Add index
DROP INDEX ip_addr_idx;

DROP EXTENSION ip4r;

COMMIT;
