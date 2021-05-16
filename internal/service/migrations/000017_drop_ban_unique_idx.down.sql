begin;

CREATE UNIQUE INDEX ban_steam_id_uindex ON ban (steam_id);
DROP INDEX IF EXISTS ban_valid_until_index;

commit;
