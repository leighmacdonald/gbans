begin;

-- Dedicated channel for chat filter warning notices. Empty falls back to the
-- word filter log channel.
alter table if exists config
    add column if not exists discord_chat_filter_warn_log_channel_id text not null default '';

commit;
