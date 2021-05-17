begin;

ALTER TABLE person ADD COLUMN IF NOT EXISTS community_banned bool not null default false;
ALTER TABLE person ADD COLUMN IF NOT EXISTS vac_bans int not null default 0;
ALTER TABLE person ADD COLUMN IF NOT EXISTS game_bans int not null default 0;
ALTER TABLE person ADD COLUMN IF NOT EXISTS economy_ban varchar not null default '';
ALTER TABLE person ADD COLUMN IF NOT EXISTS days_since_last_ban int not null default 0;

commit;
