BEGIN;

DROP FUNCTION steam_to_steam64(steam2 text);
DROP FUNCTION check_ban(steam2 text, ip text);
DROP TABLE steam_group_members;
DROP TABLE cidr_block_entries;

COMMIT;
