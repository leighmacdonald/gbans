BEGIN;

alter table forum_thread
    drop column last_forum_message_id;

alter table wiki add column permission_level int default 1 not null;

COMMIT;
