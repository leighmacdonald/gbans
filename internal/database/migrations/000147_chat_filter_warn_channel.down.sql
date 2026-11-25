begin;

alter table if exists config
    drop column if exists discord_chat_filter_warn_log_channel_id;

commit;
