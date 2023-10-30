BEGIN;

drop table if exists members;

ALTER TABLE IF EXISTS ban
    DROP COLUMN IF EXISTS include_friends;

CREATE UNIQUE INDEX ban_group_group_id_uindex ON ban_group (group_id);

COMMIT;
