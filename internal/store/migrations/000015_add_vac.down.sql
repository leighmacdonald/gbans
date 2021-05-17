begin;

ALTER TABLE person DROP COLUMN IF EXISTS community_banned;
ALTER TABLE person DROP COLUMN IF EXISTS vac_bans;
ALTER TABLE person DROP COLUMN IF EXISTS game_bans;
ALTER TABLE person DROP COLUMN IF EXISTS economy_banned;
ALTER TABLE person DROP COLUMN IF EXISTS days_since_last_ban;

commit;
