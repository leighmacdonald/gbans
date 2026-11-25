begin;

alter table if exists config
    drop column if exists discord_chat_log_enabled;

commit;
