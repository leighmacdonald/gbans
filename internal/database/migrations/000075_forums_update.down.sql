BEGIN;

ALTER TABLE forum_thread ADD COLUMN last_forum_message_id bigint;

ALTER TABLE forum_thread
    ADD CONSTRAINT fk_last_thread_id
        FOREIGN KEY (last_forum_message_id) REFERENCES forum_message (forum_message_id) ON DELETE CASCADE;

ALTER TABLE wiki DROP COLUMN permission_level;

COMMIT;
