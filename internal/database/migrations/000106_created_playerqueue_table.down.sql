BEGIN;

DROP TABLE IF EXISTS playerqueue_messages;

ALTER TABLE config
    DROP COLUMN IF EXISTS general_playerqueue_enabled;

ALTER TABLE config
    DROP COLUMN IF EXISTS discord_playerqueue_channel_id;

ALTER TABLE person
    DROP COLUMN IF EXISTS playerqueue_chat_reason;

ALTER TABLE person
    DROP COLUMN IF EXISTS playerqueue_chat_status;

DROP TYPE IF EXISTS chat_status;

COMMIT;
