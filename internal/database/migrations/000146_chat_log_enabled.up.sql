begin;

-- Controls whether in-game chat/connect messages are forwarded to the Discord
-- chat log channel. Enabled by default to preserve existing behaviour.
alter table if exists config
    add column if not exists discord_chat_log_enabled bool not null default true;

commit;
