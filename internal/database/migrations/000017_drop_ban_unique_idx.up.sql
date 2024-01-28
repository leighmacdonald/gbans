begin;

DROP INDEX IF EXISTS ban_steam_id_uindex;
CREATE INDEX IF NOT EXISTS ban_valid_until_index ON ban (valid_until);

commit;
