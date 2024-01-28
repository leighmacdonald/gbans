BEGIN;

ALTER TABLE IF EXISTS person_messages
    DROP COLUMN IF EXISTS name_search;
ALTER TABLE IF EXISTS person_messages
    DROP COLUMN IF EXISTS message_search;

DROP INDEX IF EXISTS idx_message_search;
DROP INDEX IF EXISTS idx_message_steam_id;
DROP INDEX IF EXISTS idx_created;

COMMIT;
