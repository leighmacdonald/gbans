BEGIN;

alter table forum_thread
    drop column last_forum_message_id;

COMMIT;
